package portal

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"srun-auth/config"
	"srun-auth/util"
)

type State int

const (
	Unknown State = iota
	Online
	Offline
)

func Status(cfg *config.Config) (State, error) {
	return StatusContext(context.Background(), cfg)
}

// StatusContext 查询当前请求来源，不携带可能已过期的客户端 IP。
func StatusContext(ctx context.Context, cfg *config.Config) (State, error) {
	endpoint := cfg.StatusURL
	if endpoint == "" {
		address := strings.TrimSpace(cfg.PortalIP)
		if !strings.Contains(address, "://") {
			address = "http://" + address
		}
		u, err := url.Parse(address)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return Unknown, fmt.Errorf("持续监测需要将 portal_ip 配置为实际 SRun 门户地址")
		}
		endpoint = u.Scheme + "://" + u.Host + "/cgi-bin/rad_user_info"
	}
	timestamp := util.GetCurrentTimeMillis()
	response, err := requestJSONP(ctx, endpoint, url.Values{
		"callback": {"jQuery_" + timestamp},
		"_":        {timestamp},
	}, cfg.UserAgent)
	if err != nil {
		return Unknown, err
	}
	switch response.Error {
	case "ok":
		return Online, nil
	case "not_online_error":
		return Offline, nil
	default:
		return Unknown, response.failure()
	}
}
