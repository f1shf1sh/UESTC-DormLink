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

func TestRedirectDiscoversPortal(t *testing.T) {
	for _, tt := range []struct {
		name     string
		entry    string
		pageURL  string
		pageBody string
		wantIP   string
	}{
		{"one HTTP redirect", "one", "/srun_portal_pc?ac_id=3&wlanuserip=10.0.1.1", "", "10.0.1.1"},
		{"two HTTP redirects", "two", "/srun_portal_pc?ac_id=3&wlanuserip=10.0.1.1", "", "10.0.1.1"},
		{"meta refresh", "meta", "/srun_portal_pc?ac_id=3&wlanuserip=10.0.1.1", "", "10.0.1.1"},
		{"HTML configuration", "one", "/srun_portal_pc", `<script>var CONFIG = {"acid":"3","ip":"10.0.1.1"};</script>`, "10.0.1.1"},
		{"IP input", "one", "/srun_portal_pc?ac_id=3", `<input type="hidden" id="user_ip" value="10.0.1.1">`, "10.0.1.1"},
		{"IP deferred until challenge", "one", "/srun_portal_pc?ac_id=3", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					switch tt.entry {
					case "two":
						http.Redirect(w, r, "/intermediate", http.StatusFound)
					case "meta":
						fmt.Fprintf(w, `<html><meta http-equiv="refresh" content="0; url=%s"></html>`, strings.ReplaceAll(tt.pageURL, "&", "&amp;"))
					default:
						http.Redirect(w, r, tt.pageURL, http.StatusFound)
					}
				case "/intermediate":
					http.Redirect(w, r, tt.pageURL, http.StatusFound)
				case "/srun_portal_pc":
					fmt.Fprint(w, `<html><script src="/static/themes/pro/js/Portal.js"></script>`+tt.pageBody+`</html>`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			cfg := &config.Config{PortalIP: server.URL}
			if err := portal.Redirect(cfg); err != nil {
				t.Fatalf("Redirect: %v", err)
			}
			if cfg.ACID != "3" || cfg.OnlineIP != tt.wantIP {
				t.Errorf("ac_id/IP = %q/%q, want 3/%q", cfg.ACID, cfg.OnlineIP, tt.wantIP)
			}
			if want := server.URL + "/cgi-bin/srun_portal"; cfg.AuthURL != want {
				t.Errorf("AuthURL = %q, want %q", cfg.AuthURL, want)
			}
			if want := server.URL + "/cgi-bin/get_challenge"; cfg.ChallengeURL != want {
				t.Errorf("ChallengeURL = %q, want %q", cfg.ChallengeURL, want)
			}
		})
	}
}

func TestRedirectRejectsUnsupportedPages(t *testing.T) {
	for _, tt := range []struct {
		name string
		path string
		body string
	}{
		{"missing ac_id", "/srun_portal_pc", `<html>srun portal</html>`},
		{"ePortal", "/eportal/index.jsp?ac_id=3", `<html>ePortal</html>`},
		{"unrelated page", "/index.html", `<html>ordinary website</html>`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/" {
					http.Redirect(w, r, tt.path, http.StatusFound)
					return
				}
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			if err := portal.Redirect(&config.Config{PortalIP: server.URL}); err == nil {
				t.Fatal("Redirect accepted an unsupported or incomplete portal page")
			}
		})
	}
}
