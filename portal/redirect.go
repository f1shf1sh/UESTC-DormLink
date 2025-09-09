package portal

import (
	"net/http"
	"net/url"
	"srun-auth/config"
)

func Redirect(cfg *config.Config) error {
	portal_url := "http://" + cfg.PortalIP
	req, err := http.NewRequest("GET", portal_url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 返回 http.ErrUseLastResponse 来阻止重定向
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	location, err := resp.Location()
	if err != nil {
		return err
	}

	redirectURL := location.String()
	req2, err := http.NewRequest("GET", redirectURL, nil)
	if err != nil {
		return err
	}

	resp2, err := client.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	location, err = resp2.Location()
	if err != nil {
		return err
	}
	loginURL := location.String()
	//解析登陆界面的url
	u, err := url.Parse(loginURL)
	if err != nil {
		return err
	}

	host_ip := u.Host
	cfg.ChallengeURL = "http://" + host_ip + "/cgi-bin/get_challenge"
	cfg.AuthURL = "http://" + host_ip + "/cgi-bin/srun_portal_pc"

	query_params := u.Query()
	cfg.OnlineIP = query_params.Get("wlanuserip")
	cfg.ACID = query_params.Get("ac_id")
	cfg.WlanACIP = query_params.Get("wlanacip")
	return nil
}
