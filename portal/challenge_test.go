package portal_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"srun-auth/config"
	"srun-auth/portal"
)

func TestLoginWithoutCarrierUsesBareUsername(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if got := r.URL.Query().Get("username"); got != "111111" {
			t.Errorf("%s username = %q, want 111111", r.URL.Path, got)
		}
		switch r.URL.Path {
		case "/cgi-bin/get_challenge":
			writeJSONP(w, r, fmt.Sprintf(`{"error":"ok","challenge":%q,"client_ip":"10.0.1.1"}`, testToken))
		case "/cgi-bin/srun_portal":
			writeJSONP(w, r, `{"error":"ok","suc_msg":"login_ok"}`)
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cfg := &config.Config{
		Username: "111111", Password: "12345", ACID: "3",
		ChallengeURL: server.URL + "/cgi-bin/get_challenge",
		AuthURL:      server.URL + "/cgi-bin/srun_portal",
	}
	if err := portal.GetChallenge(cfg); err != nil {
		t.Fatalf("GetChallenge: %v", err)
	}
	if err := portal.Auth(cfg); err != nil {
		t.Fatalf("Auth: %v", err)
	}
	if got := strings.Join(paths, ","); got != "/cgi-bin/get_challenge,/cgi-bin/srun_portal" {
		t.Errorf("request sequence = %q", got)
	}
}

func TestGetChallenge(t *testing.T) {
	for _, tt := range []struct {
		name    string
		body    string
		inputIP string
		wantIP  string
		wantErr bool
	}{
		{"server supplies IP", `{"error":"ok","challenge":"token","client_ip":"10.0.1.1"}`, "", "10.0.1.1", false},
		{"server replaces stale IP", `{"error":"ok","challenge":"token","client_ip":"10.0.1.1"}`, "10.0.1.8", "10.0.1.1", false},
		{"retain page IP", `{"error":"ok","challenge":"token"}`, "10.0.1.1", "10.0.1.1", false},
		{"missing token", `{"error":"ok","client_ip":"10.0.1.1"}`, "10.0.1.1", "", true},
		{"missing IP", `{"error":"ok","challenge":"token"}`, "", "", true},
		{"server rejection", `{"error":"E2531","error_msg":"invalid user"}`, "10.0.1.1", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeJSONP(w, r, tt.body)
			}))
			defer server.Close()
			cfg := &config.Config{Username: "111111", OnlineIP: tt.inputIP, ChallengeURL: server.URL}
			err := portal.GetChallenge(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("GetChallenge reported success for an unusable challenge")
				}
				if tt.name == "server rejection" && (!strings.Contains(err.Error(), "E2531") || !strings.Contains(err.Error(), "invalid user")) {
					t.Errorf("error does not explain server rejection: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetChallenge: %v", err)
			}
			if cfg.Token != "token" || cfg.OnlineIP != tt.wantIP {
				t.Errorf("token/IP = %q/%q, want token/%q", cfg.Token, cfg.OnlineIP, tt.wantIP)
			}
		})
	}
}
