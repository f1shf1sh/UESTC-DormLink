package portal_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"srun-auth/config"
	"srun-auth/portal"
)

func TestStatusUsesCurrentGatewayWithoutLoginParameters(t *testing.T) {
	wrongGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("status queried AuthURL's gateway: %s", r.URL)
		http.Error(w, "wrong gateway", http.StatusBadRequest)
	}))
	defer wrongGateway.Close()
	for _, tt := range []struct {
		name     string
		bareHost bool
		explicit bool
	}{
		{name: "portal URL origin"},
		{name: "bare portal host", bareHost: true},
		{name: "discovered status URL takes priority", explicit: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			wantPath := "/cgi-bin/rad_user_info"
			if tt.explicit {
				wantPath = "/discovered/cgi-bin/rad_user_info"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != wantPath {
					t.Errorf("status request = %s %s, want GET %s", r.Method, r.URL.Path, wantPath)
				}
				q := r.URL.Query()
				if len(q) != 2 || q.Get("callback") == "" || q.Get("_") == "" {
					t.Errorf("status must send only callback and timestamp, got %v", q)
				}
				if r.UserAgent() != "srun-status-test" {
					t.Errorf("User-Agent = %q", r.UserAgent())
				}
				writeJSONP(w, r, `{"error":"ok","online_ip":"10.0.1.9"}`)
			}))
			defer server.Close()
			cfg := &config.Config{
				PortalIP: server.URL + "/srun_portal_pc?ac_id=3",
				AuthURL:  wrongGateway.URL + "/cgi-bin/srun_portal",
				Username: "old-user", Password: "old-password", Carrier: "dx",
				OnlineIP: "10.0.1.8", Token: "old-token", UserAgent: "srun-status-test",
			}
			if tt.bareHost {
				cfg.PortalIP = strings.TrimPrefix(server.URL, "http://")
			}
			if tt.explicit {
				cfg.StatusURL = server.URL + wantPath
				cfg.PortalIP = wrongGateway.URL
			}
			state, err := portal.Status(cfg)
			if err != nil || state != portal.Online {
				t.Fatalf("Status = %v, %v; want Online, nil", state, err)
			}
			if got := requests.Load(); got != 1 {
				t.Errorf("status made %d requests; an online client needs only one status request", got)
			}
		})
	}
}

func TestStatusDistinguishesOfflineFromUnknown(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		body    string
		jsonp   bool
		want    portal.State
		wantErr bool
	}{
		{"online", 200, `{"error":"ok"}`, true, portal.Online, false},
		{"offline", 200, `{"error":"not_online_error"}`, true, portal.Offline, false},
		{"business failure", 200, `{"error":"backend_unavailable","error_msg":"try later"}`, true, portal.Unknown, true},
		{"missing error", 200, `{"online_ip":"10.0.1.1"}`, true, portal.Unknown, true},
		{"invalid JSON", 200, `{"error":`, true, portal.Unknown, true},
		{"wrong callback", 200, `another_callback({"error":"not_online_error"})`, false, portal.Unknown, true},
		{"login HTML", 200, `<html>not_online_error</html>`, false, portal.Unknown, true},
		{"HTTP failure", 503, `service unavailable`, false, portal.Unknown, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.jsonp {
					writeJSONP(w, r, tt.body)
				} else {
					fmt.Fprint(w, tt.body)
				}
			}))
			defer server.Close()
			state, err := portal.Status(&config.Config{PortalIP: server.URL})
			if state != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("Status = %v, %v; want state %v, error %t", state, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestStatusConnectionFailureIsUnknown(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	state, err := portal.Status(&config.Config{PortalIP: server.URL})
	if state != portal.Unknown || err == nil {
		t.Errorf("Status = %v, %v; want Unknown and a connection error", state, err)
	}
}

func TestStatusContextCancelsPendingRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		state portal.State
		err   error
	}
	done := make(chan result, 1)
	go func() {
		state, err := portal.StatusContext(ctx, &config.Config{PortalIP: server.URL})
		done <- result{state, err}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("status request did not reach the server")
	}
	cancel()
	select {
	case got := <-done:
		if got.state != portal.Unknown || !errors.Is(got.err, context.Canceled) {
			t.Errorf("canceled StatusContext = %v, %v; want Unknown and context.Canceled", got.state, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("StatusContext did not return promptly after cancellation")
	}
}
