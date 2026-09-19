package portal_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"srun-auth/config"
	"srun-auth/portal"
)

func TestLoginRefreshesPortalIPAndChallengeAfterDisconnection(t *testing.T) {
	var mu sync.Mutex
	var requests []string
	var passwords []string
	loginRound, statusCount := 0, 0
	actualGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, "gateway:"+r.URL.Path)
		switch r.URL.Path {
		case "/srun_portal_pc":
			fmt.Fprint(w, `<html><script src="/static/themes/pro/js/Portal.js"></script></html>`)
		case "/cgi-bin/get_challenge":
			q := r.URL.Query()
			wantIP := fmt.Sprintf("10.0.%d.8", loginRound)
			if q.Get("username") != "111111@dx" || q.Get("ip") != wantIP {
				t.Errorf("challenge username/IP = %q/%q, want 111111@dx/%s", q.Get("username"), q.Get("ip"), wantIP)
			}
			writeJSONP(w, r, fmt.Sprintf(`{"error":"ok","challenge":"fresh-token-%d","client_ip":"10.0.%d.1"}`, loginRound, loginRound))
		case "/cgi-bin/srun_portal":
			q := r.URL.Query()
			if got, want := q.Get("ip"), fmt.Sprintf("10.0.%d.1", loginRound); got != want {
				t.Errorf("auth IP = %q, want new challenge IP %q", got, want)
			}
			if q.Get("username") != "111111@dx" || q.Get("ac_id") != "3" {
				t.Errorf("auth username/ac_id = %q/%q", q.Get("username"), q.Get("ac_id"))
			}
			passwords = append(passwords, q.Get("password"))
			writeJSONP(w, r, `{"error":"ok","suc_msg":"login_ok"}`)
		case "/cgi-bin/rad_user_info":
			q := r.URL.Query()
			if len(q) != 2 || q.Get("callback") == "" || q.Get("_") == "" {
				t.Errorf("status sent stale login parameters: %v", q)
			}
			statusCount++
			if statusCount == 2 {
				writeJSONP(w, r, `{"error":"not_online_error"}`)
			} else {
				writeJSONP(w, r, `{"error":"ok"}`)
			}
		default:
			t.Errorf("unexpected gateway request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer actualGateway.Close()
	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, "entry:"+r.URL.Path)
		switch r.URL.Path {
		case "/cgi-bin/rad_user_info":
			writeJSONP(w, r, `{"error":"not_online_error"}`)
		case "/":
			loginRound++
			location := fmt.Sprintf("%s/srun_portal_pc?ac_id=3&wlanuserip=10.0.%d.8", actualGateway.URL, loginRound)
			http.Redirect(w, r, location, http.StatusFound)
		default:
			t.Errorf("unexpected entry request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer entry.Close()
	cfg := &config.Config{
		PortalIP: entry.URL, Username: "111111", Password: "12345", Carrier: "dx",
		Token: "stale-token", OnlineIP: "10.0.0.1",
	}
	for round := 1; round <= 2; round++ {
		if state, err := portal.Status(cfg); err != nil || state != portal.Offline {
			t.Fatalf("round %d initial Status = %v, %v; want Offline", round, state, err)
		}
		if err := portal.Login(cfg); err != nil {
			t.Fatalf("round %d Login: %v", round, err)
		}
		if got, want := cfg.Token, fmt.Sprintf("fresh-token-%d", round); got != want {
			t.Errorf("round %d token = %q, want %q", round, got, want)
		}
		if got, want := cfg.OnlineIP, fmt.Sprintf("10.0.%d.1", round); got != want {
			t.Errorf("round %d IP = %q, want %q", round, got, want)
		}
		if want := actualGateway.URL + "/cgi-bin/rad_user_info"; cfg.StatusURL != want {
			t.Errorf("StatusURL = %q, want discovered gateway %q", cfg.StatusURL, want)
		}
		if state, err := portal.Status(cfg); err != nil || state != portal.Online {
			t.Fatalf("round %d final Status = %v, %v; want Online", round, state, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(passwords) != 2 || passwords[0] == "" || passwords[0] == passwords[1] {
		t.Errorf("separate login attempts did not use different challenge-derived passwords: %v", passwords)
	}
	wantRequests := []string{
		"entry:/cgi-bin/rad_user_info",
		"entry:/", "gateway:/srun_portal_pc", "gateway:/cgi-bin/get_challenge", "gateway:/cgi-bin/srun_portal",
		"gateway:/cgi-bin/rad_user_info", "gateway:/cgi-bin/rad_user_info",
		"entry:/", "gateway:/srun_portal_pc", "gateway:/cgi-bin/get_challenge", "gateway:/cgi-bin/srun_portal",
		"gateway:/cgi-bin/rad_user_info",
	}
	if got, want := strings.Join(requests, ","), strings.Join(wantRequests, ","); got != want {
		t.Errorf("request sequence = %s, want %s", got, want)
	}
}

func TestLoginContextCancelsChallengeWithoutSendingAuth(t *testing.T) {
	challengeStarted := make(chan struct{})
	release := make(chan struct{})
	var authRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/srun_portal_pc?ac_id=3&wlanuserip=10.0.1.1", http.StatusFound)
		case "/srun_portal_pc":
			fmt.Fprint(w, `<html>srun portal</html>`)
		case "/cgi-bin/get_challenge":
			close(challengeStarted)
			select {
			case <-r.Context().Done():
			case <-release:
			}
		case "/cgi-bin/srun_portal":
			authRequests.Add(1)
			writeJSONP(w, r, `{"error":"ok"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := &config.Config{
		PortalIP: server.URL, Username: "111111", Password: "12345", Carrier: "dx",
		Token: "stale-token", OnlineIP: "10.0.0.1",
	}
	done := make(chan error, 1)
	go func() { done <- portal.LoginContext(ctx, cfg) }()
	select {
	case <-challengeStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("login did not reach the challenge request")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("canceled LoginContext = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("LoginContext did not return promptly after cancellation")
	}
	if got := authRequests.Load(); got != 0 {
		t.Errorf("canceled login sent %d authentication requests", got)
	}
	if cfg.Token != "" {
		t.Errorf("canceled challenge retained a stale token: %q", cfg.Token)
	}
}

func TestIsPermanentClassifiesOnlyConfirmedBusinessErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{"account locked", &portal.ResponseError{Code: "account_locked"}, true},
		{"password change required", &portal.ResponseError{Message: "user_must_modify_password"}, true},
		{"nonexistent user", &portal.ResponseError{ECode: "E2531"}, true},
		{"invalid credentials", &portal.ResponseError{ECode: "E2553"}, true},
		{"credential code in error", &portal.ResponseError{Code: "E2553"}, true},
		{"user code in error message", &portal.ResponseError{Message: "E2531"}, true},
		{"user message prefix", &portal.ResponseError{Message: "E2531: User does not exist"}, true},
		{"password message prefix", &portal.ResponseError{Message: "E2553: Password Error"}, true},
		{"exact policy code", &portal.ResponseError{PolicyMessage: "account_locked"}, true},
		{"policy message prefix", &portal.ResponseError{PolicyMessage: "E2553: Password Error"}, true},
		{"unconfirmed E2901", &portal.ResponseError{Code: "E2901"}, false},
		{"unconfirmed E2620", &portal.ResponseError{ECode: "E2620"}, false},
		{"longer code", &portal.ResponseError{Message: "E25530: unrelated error"}, false},
		{"code after other text", &portal.ResponseError{PolicyMessage: "upstream replied E2553: Password Error"}, false},
		{"partial code text", &portal.ResponseError{Message: "received E2553 from another service"}, false},
		{"network error", errors.New("connection refused"), false},
		{"empty business error", &portal.ResponseError{}, false},
		{"nil error", nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := portal.IsPermanent(tt.err); got != tt.want {
				t.Errorf("IsPermanent(%v) = %t, want %t", tt.err, got, tt.want)
			}
			if tt.err != nil {
				wrapped := fmt.Errorf("authentication failed: %w", tt.err)
				if got := portal.IsPermanent(wrapped); got != tt.want {
					t.Errorf("IsPermanent(wrapped %v) = %t, want %t", tt.err, got, tt.want)
				}
			}
		})
	}
}

func TestAuthRecognizesCredentialCodesInPortalMessages(t *testing.T) {
	for _, tt := range []struct {
		name    string
		payload string
	}{
		{"error message", `{"error":"login_error","error_msg":"E2553: Password Error"}`},
		{"policy message", `{"error":"login_error","ploy_msg":"E2553: Password Error"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeJSONP(w, r, tt.payload)
			}))
			defer server.Close()
			err := portal.Auth(&config.Config{
				Username: "111111", Password: "12345", Carrier: "dx",
				ACID: "3", OnlineIP: "10.0.1.1", Token: testToken,
				AuthURL: server.URL + "/cgi-bin/srun_portal",
			})
			var response *portal.ResponseError
			if !errors.As(err, &response) {
				t.Fatalf("Auth error = %v, want ResponseError", err)
			}
			if response.Code != "login_error" || response.ECode != "" {
				t.Errorf("business error codes = %q/%q, want login_error and no ecode", response.Code, response.ECode)
			}
			if !portal.IsPermanent(err) {
				t.Errorf("credential error in portal message was not classified as permanent: %v", err)
			}
		})
	}
}
