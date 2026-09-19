package portal

import (
	"context"
	"fmt"
	"net/url"
	"srun-auth/config"
	"srun-auth/util"
)

func GetChallenge(cfg *config.Config) error {
	return getChallenge(context.Background(), cfg)
}

func getChallenge(ctx context.Context, cfg *config.Config) error {
	cfg.Token = ""
	timestamp := util.GetCurrentTimeMillis()
	params := url.Values{
		"callback": {"jQuery_" + timestamp},
		"username": {cfg.LoginUsername()},
		"ip":       {cfg.OnlineIP},
		"_":        {timestamp},
	}
	response, err := getJSONP(ctx, cfg.ChallengeURL, params, cfg.UserAgent)
	if err != nil {
		return err
	}
	if response.Challenge == "" {
		return fmt.Errorf("服务器未返回 challenge")
	}
	// 当前客户端登录本机网络，优先采用认证服务器观察到的地址。
	if response.ClientIP != "" {
		cfg.OnlineIP = response.ClientIP
	}
	if cfg.OnlineIP == "" {
		return fmt.Errorf("门户和 challenge 均未提供客户端 IP")
	}
	cfg.Token = response.Challenge
	return nil
}
