package portal

import (
	"context"
	"fmt"

	"srun-auth/config"
)

func Login(cfg *config.Config) error {
	return LoginContext(context.Background(), cfg)
}

// LoginContext 每次重新发现网络参数并取得 challenge，不重放旧认证请求。
func LoginContext(ctx context.Context, cfg *config.Config) error {
	cfg.Token, cfg.OnlineIP, cfg.ACID = "", "", ""
	if err := redirect(ctx, cfg); err != nil {
		return fmt.Errorf("发现认证门户失败: %w", err)
	}
	if err := getChallenge(ctx, cfg); err != nil {
		return fmt.Errorf("获取 challenge 失败: %w", err)
	}
	if err := auth(ctx, cfg); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}
	return nil
}
