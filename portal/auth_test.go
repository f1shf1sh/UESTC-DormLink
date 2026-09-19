package portal_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"srun-auth/config"
	"srun-auth/portal"
)

// These values use fictitious credentials. The encoded info is independently
// generated from the reference portal implementation, not the code under test.
const (
	testToken = "7f31ebcccbc14318005e0ba573ce3e681f0ae52c20e156bf6520f209b68308b7"
	testInfo  = "{SRBX1}sL2W5810S9ESa7V96wCXmUQc51xM7RhXGndKke3TCXKXQopHTetb5tmQEEkkjQAdswAB1fN3QYdt9FGSfmuXv/JU/3AjtkLdoaRNGzPw//Wz8I5GVS406tSQtMZLayVJ"
)

func writeJSONP(w http.ResponseWriter, r *http.Request, payload string) {
	w.Header().Set("Content-Type", "application/javascript")
	fmt.Fprintf(w, "%s(%s)", r.URL.Query().Get("callback"), payload)
}

func TestLoginFlow(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if got := r.UserAgent(); got != "srun-auth-test" {
			t.Errorf("User-Agent = %q, want srun-auth-test", got)
		}
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/srun_portal_pc?ac_id=3&wlanuserip=10.0.1.8", http.StatusFound)
		case "/srun_portal_pc":
			fmt.Fprint(w, `<html><script src="/static/themes/pro/js/Portal.js"></script></html>`)
		case "/cgi-bin/get_challenge":
			q := r.URL.Query()
			if q.Get("username") != "111111@dx" || q.Get("ip") != "10.0.1.8" {
				t.Errorf("challenge username/ip = %q/%q", q.Get("username"), q.Get("ip"))
			}
			if q.Get("callback") == "" || q.Get("_") == "" {
				t.Error("challenge is missing callback or timestamp")
			}
			writeJSONP(w, r, fmt.Sprintf(`{"error":"ok","challenge":%q,"client_ip":"10.0.1.1"}`, testToken))
		case "/cgi-bin/srun_portal":
			q := r.URL.Query()
			want := map[string]string{
				"action":       "login",
				"username":     "111111@dx",
				"password":     "{MD5}b6983d1d5d9cac868bf0015069671cf2",
				"ac_id":        "3",
				"ip":           "10.0.1.1",
				"info":         testInfo,
				"chksum":       "4806f2e3f187e120887a93d6fc27e8e944ff747a",
				"n":            "200",
				"type":         "1",
				"double_stack": "0",
			}
			for key, value := range want {
				if got := q.Get(key); got != value {
					t.Errorf("login parameter %s = %q, want %q", key, got, value)
				}
			}
			if q.Get("callback") == "" || q.Get("_") == "" {
				t.Error("login is missing callback or timestamp")
			}
			writeJSONP(w, r, `{"error":"ok","suc_msg":"login_ok"}`)
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cfg := &config.Config{
		Username: "111111", Password: "12345", Carrier: "dx",
		PortalIP: server.URL, UserAgent: "srun-auth-test",
	}
	if err := portal.Redirect(cfg); err != nil {
		t.Fatalf("Redirect: %v", err)
	}
	if err := portal.GetChallenge(cfg); err != nil {
		t.Fatalf("GetChallenge: %v", err)
	}
	if err := portal.Auth(cfg); err != nil {
		t.Fatalf("Auth: %v", err)
	}
	wantPaths := "/,/srun_portal_pc,/cgi-bin/get_challenge,/cgi-bin/srun_portal"
	if got := strings.Join(paths, ","); got != wantPaths {
		t.Errorf("request sequence = %q, want %q", got, wantPaths)
	}
}

func TestAuthRejectsFailedResponses(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		jsonp  bool
		want   []string
	}{
		{"rejected credentials", 200, `{"error":"E2553","error_msg":"invalid account"}`, true, []string{"E2553", "invalid account"}},
		{"missing error field", 200, `{"suc_msg":"login_ok"}`, true, nil},
		{"malformed JSON", 200, `{"error":"ok",`, true, nil},
		{"wrong callback", 200, `unexpected_callback({"error":"ok"})`, false, nil},
		{"HTML instead of JSONP", 200, `<html>login required</html>`, false, nil},
		{"HTTP failure", 503, `service unavailable`, false, []string{"503"}},
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
			cfg := &config.Config{
				Username: "111111", Password: "12345", Carrier: "dx",
				ACID: "3", OnlineIP: "10.0.1.1", Token: testToken,
				AuthURL: server.URL + "/cgi-bin/srun_portal",
			}
			err := portal.Auth(cfg)
			if err == nil {
				t.Fatal("Auth reported success for a failed response")
			}
			for _, part := range tt.want {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q does not contain %q", err, part)
				}
			}
		})
	}
}

func TestRequestConnectionErrorsDoNotExposeCredentials(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	for _, tt := range []struct {
		name string
		call func(*config.Config) error
	}{
		{"challenge", portal.GetChallenge},
		{"authentication", portal.Auth},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Username: "111111", Password: "12345", Carrier: "dx",
				ACID: "3", OnlineIP: "10.0.1.1", Token: testToken,
				AuthURL:      server.URL + "/cgi-bin/srun_portal",
				ChallengeURL: server.URL + "/cgi-bin/get_challenge",
			}
			err := tt.call(cfg)
			if err == nil {
				t.Fatal("request to a closed server unexpectedly succeeded")
			}
			for _, secret := range []string{
				server.URL, cfg.Username, cfg.Password, testToken, testInfo,
				"b6983d1d5d9cac868bf0015069671cf2", "username=", "password=", "info=",
			} {
				if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), url.QueryEscape(secret)) {
					t.Errorf("connection error exposes request credentials: %v", err)
				}
			}
		})
	}
}
