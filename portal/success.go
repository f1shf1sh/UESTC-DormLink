package portal

import (
	"net/http"
	"net/url"

	"srun-auth/config"
)

func success(cfg *config.Config) error {
	params := &url.Values{}
	params.Add("ac_id", cfg.ACID)
	params.Add("theme", "yd")
	params.Add("wlanacip", cfg.WlanACIP)
	params.Add("wlanacname", "")
	params.Add("wlanuserip", cfg.OnlineIP)
	params.Add("srun_domain", "@"+cfg.Carrier)
}
