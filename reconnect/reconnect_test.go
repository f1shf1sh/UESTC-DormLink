package reconnect

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"srun-auth/config"
	"srun-auth/portal"
)

func testRunner() *runner {
	return &runner{
		cfg:       &config.Config{},
		logf:      func(string, ...any) {},
		permanent: func(error) bool { return false },
	}
}

func TestNoLoginWithoutConfirmedOffline(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state portal.State
		err   error
	}{
		{"online", portal.Online, nil},
		{"unknown", portal.Unknown, nil},
		{"request failure", portal.Unknown, errors.New("timeout")},
		{"error with offline state", portal.Offline, errors.New("invalid response")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := testRunner()
			r.status = func(context.Context, *config.Config) (portal.State, error) {
				return tc.state, tc.err
			}
			r.login = func(context.Context, *config.Config) error {
				t.Fatal("login called without a confirmed offline response")
				return nil
			}
			r.step(context.Background())
		})
	}
}

func TestReconnectAndConfirmEveryDisconnect(t *testing.T) {
	r := testRunner()
	var calls []string
	statusCalls, loginCalls := 0, 0
	r.status = func(_ context.Context, cfg *config.Config) (portal.State, error) {
		statusCalls++
		calls = append(calls, "status")
		if statusCalls%2 == 1 {
			return portal.Offline, nil
		}
		if cfg.Token != fmt.Sprintf("fresh-token-%d", loginCalls) {
			t.Fatal("status confirmation did not observe the latest login context")
		}
		return portal.Online, nil
	}
	r.login = func(_ context.Context, cfg *config.Config) error {
		loginCalls++
		calls = append(calls, "login")
		cfg.Token = fmt.Sprintf("fresh-token-%d", loginCalls)
		return nil
	}
	r.step(context.Background())
	r.step(context.Background())
	want := []string{"status", "login", "status", "status", "login", "status"}
	if !reflect.DeepEqual(calls, want) || loginCalls != 2 {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestAcceptedRequestStillRequiresOnlineConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state portal.State
		err   error
	}{
		{"still offline", portal.Offline, nil},
		{"unknown", portal.Unknown, nil},
		{"query failure", portal.Unknown, errors.New("timeout")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := testRunner()
			var logs []string
			r.logf = func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }
			calls := 0
			r.status = func(context.Context, *config.Config) (portal.State, error) {
				calls++
				if calls == 1 {
					return portal.Offline, nil
				}
				return tc.state, tc.err
			}
			r.login = func(context.Context, *config.Config) error { return nil }
			r.step(context.Background())
			if r.state == portal.Online || !strings.Contains(strings.Join(logs, "\n"), "认证请求已接受，但") {
				t.Fatalf("unconfirmed login reported incorrectly: state=%v logs=%v", r.state, logs)
			}
		})
	}
}

func TestTemporaryLoginFailureRetriesNextCheck(t *testing.T) {
	r := testRunner()
	statusCalls, loginCalls := 0, 0
	r.status = func(context.Context, *config.Config) (portal.State, error) {
		statusCalls++
		return portal.Offline, nil
	}
	r.login = func(context.Context, *config.Config) error {
		loginCalls++
		return errors.New("temporary failure")
	}
	r.step(context.Background())
	r.step(context.Background())
	if loginCalls != 2 || statusCalls != 2 || r.paused {
		t.Fatalf("status=%d login=%d paused=%v", statusCalls, loginCalls, r.paused)
	}
}

func TestPermanentRejectionPausesUntilStopped(t *testing.T) {
	r := testRunner()
	accountError := errors.New("account locked")
	statusCalls, loginCalls := 0, 0
	r.status = func(context.Context, *config.Config) (portal.State, error) {
		statusCalls++
		return portal.Offline, nil
	}
	r.login = func(context.Context, *config.Config) error {
		loginCalls++
		return accountError
	}
	r.permanent = func(err error) bool { return errors.Is(err, accountError) }
	r.step(context.Background())
	r.step(context.Background())
	if !r.paused || statusCalls != 1 || loginCalls != 1 {
		t.Fatalf("status=%d login=%d paused=%v", statusCalls, loginCalls, r.paused)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- r.run(ctx, time.Millisecond) }()
	select {
	case err := <-done:
		t.Fatalf("paused process exited before cancellation: %v", err)
	default:
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cancelled run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("paused process did not stop")
	}
}

func TestRepeatedStateAndErrorAreLoggedOnce(t *testing.T) {
	r := testRunner()
	var logs []string
	r.logf = func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }
	r.status = func(context.Context, *config.Config) (portal.State, error) { return portal.Offline, nil }
	r.login = func(context.Context, *config.Config) error { return errors.New("timeout") }
	r.step(context.Background())
	r.step(context.Background())
	if len(logs) != 2 {
		t.Fatalf("repeated state/error generated repeated logs: %v", logs)
	}
	r.login = func(context.Context, *config.Config) error { return errors.New("connection refused") }
	r.step(context.Background())
	if len(logs) != 3 || !strings.Contains(logs[2], "connection refused") {
		t.Fatalf("changed error was not logged: %v", logs)
	}
	r.status = func(context.Context, *config.Config) (portal.State, error) { return portal.Online, nil }
	r.step(context.Background())
	r.status = func(context.Context, *config.Config) (portal.State, error) { return portal.Offline, nil }
	r.step(context.Background())
	if len(logs) != 6 {
		t.Fatalf("new disconnect after recovery was not reported: %v", logs)
	}
}

func TestCancellationPreventsFurtherRequests(t *testing.T) {
	for _, stage := range []string{"before check", "after status", "after login"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := testRunner()
			statusCalls, loginCalls := 0, 0
			r.status = func(context.Context, *config.Config) (portal.State, error) {
				statusCalls++
				if stage == "after status" {
					cancel()
				}
				return portal.Offline, nil
			}
			r.login = func(context.Context, *config.Config) error {
				loginCalls++
				cancel()
				return nil
			}
			if stage == "before check" {
				cancel()
			}
			r.step(ctx)
			wantStatus, wantLogin := 1, 0
			if stage == "before check" {
				wantStatus = 0
			} else if stage == "after login" {
				wantLogin = 1
			}
			if statusCalls != wantStatus || loginCalls != wantLogin {
				t.Fatalf("status=%d login=%d, want %d/%d", statusCalls, loginCalls, wantStatus, wantLogin)
			}
		})
	}
}

func TestRunStartsImmediatelyAndCancelsInFlightCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := testRunner()
	started := make(chan struct{})
	r.status = func(requestCtx context.Context, _ *config.Config) (portal.State, error) {
		close(started)
		<-requestCtx.Done()
		return portal.Unknown, requestCtx.Err()
	}
	r.login = func(context.Context, *config.Config) error {
		t.Error("login called during cancellation")
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- r.run(ctx, time.Hour) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("initial state check did not start immediately")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error on cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not cancel its request")
	}
}

func TestRunSerializesChecksAndLogin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := testRunner()
	var active atomic.Int32
	var statusCalls atomic.Int32
	var loginCalls atomic.Int32
	enter := func() {
		if active.Add(1) != 1 {
			t.Error("concurrent portal operations")
		}
	}
	r.status = func(context.Context, *config.Config) (portal.State, error) {
		enter()
		defer active.Add(-1)
		if statusCalls.Add(1) == 5 {
			cancel()
		}
		return portal.Offline, nil
	}
	r.login = func(context.Context, *config.Config) error {
		enter()
		defer active.Add(-1)
		loginCalls.Add(1)
		return errors.New("temporary failure")
	}
	if err := r.run(ctx, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if statusCalls.Load() != 5 || loginCalls.Load() != 4 {
		t.Fatalf("status=%d login=%d", statusCalls.Load(), loginCalls.Load())
	}
}

func TestRunRejectsInvalidInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		r := testRunner()
		if err := r.run(context.Background(), interval); err == nil {
			t.Fatalf("accepted interval %v", interval)
		}
	}
}

func TestRunPausesAfterPortalAccountRejection(t *testing.T) {
	var statusCalls, pageCalls, challengeCalls, authCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		var response string
		switch request.URL.Path {
		case "/cgi-bin/rad_user_info":
			statusCalls.Add(1)
			response = `{"error":"not_online_error"}`
		case "/srun_portal_pc":
			pageCalls.Add(1)
			fmt.Fprint(w, "<html>Login page</html>")
			return
		case "/cgi-bin/get_challenge":
			challengeCalls.Add(1)
			response = `{"error":"ok","challenge":"fresh-test-challenge","client_ip":"10.0.0.5"}`
		case "/cgi-bin/srun_portal":
			authCalls.Add(1)
			if query.Get("username") != "test-user@dx" || query.Get("ip") != "10.0.0.5" || query.Get("ac_id") != "3" {
				t.Error("login did not use the discovered account and network context")
			}
			response = `{"error":"login_error","ecode":"E2553","error_msg":"Account password rejected"}`
		default:
			t.Errorf("unexpected request path %q", request.URL.Path)
			http.NotFound(w, request)
			return
		}
		fmt.Fprintf(w, "%s(%s)", query.Get("callback"), response)
	}))
	defer server.Close()

	cfg := &config.Config{
		Username: "test-user",
		Password: "test-password",
		Carrier:  "dx",
		PortalIP: server.URL + "/srun_portal_pc?ac_id=3&wlanuserip=10.0.0.4",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	paused := make(chan struct{}, 1)
	logf := func(format string, args ...any) {
		if strings.Contains(fmt.Sprintf(format, args...), "已暂停重连") {
			select {
			case paused <- struct{}{}:
			default:
			}
		}
	}
	const interval = 5 * time.Millisecond
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, interval, logf) }()
	select {
	case <-paused:
	case err := <-done:
		t.Fatalf("Run exited instead of pausing: %v", err)
	case <-time.After(time.Second):
		t.Fatal("portal E2553 did not pause authentication")
	}

	// Leave the real loop enough time for several retries if the pause was ignored.
	select {
	case err := <-done:
		t.Fatalf("paused Run exited without cancellation: %v", err)
	case <-time.After(3 * interval):
	}
	if statusCalls.Load() != 1 || pageCalls.Load() != 1 || challengeCalls.Load() != 1 || authCalls.Load() != 1 {
		t.Fatalf("paused request counts: status=%d page=%d challenge=%d auth=%d",
			statusCalls.Load(), pageCalls.Load(), challengeCalls.Load(), authCalls.Load())
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cancelled Run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("paused Run did not stop after cancellation")
	}
}
