package main

import (
	"fmt"
	"os"
	"srun-auth/config"
	"srun-auth/portal"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "[-]", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadConfig("./config.json")
	if err != nil {
		return err
	}
	if err := portal.Redirect(cfg); err != nil {
		return fmt.Errorf("发现认证门户失败: %w", err)
	}
	fmt.Println("[+] 已发现认证门户")
	if err := portal.GetChallenge(cfg); err != nil {
		return fmt.Errorf("获取 challenge 失败: %w", err)
	}
	if err := portal.Auth(cfg); err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}
	fmt.Println("[+] 认证流程完成（认证服务器返回 ok）")
	return nil
}
