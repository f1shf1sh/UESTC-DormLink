package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"srun-auth/config"
	"srun-auth/portal"
	"srun-auth/reconnect"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil &&
		!errors.Is(err, flag.ErrHelp) && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "[-]", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("srun-auto-login", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("config", "./config.json", "配置文件路径")
	watch := flags.Bool("watch", false, "持续检测认证状态并在离线时重连")
	interval := flags.Duration("interval", 30*time.Second, "持续监测间隔，例如 30s 或 1m")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("不接受位置参数；使用 -help 查看选项")
	}
	if *watch && *interval <= 0 {
		return fmt.Errorf("监测间隔必须大于 0")
	}
	cfg, err := config.LoadConfig(*path)
	if err != nil {
		return err
	}
	if *watch {
		logger := log.New(stdout, "", log.LstdFlags)
		return reconnect.Run(ctx, cfg, *interval, logger.Printf)
	}
	if err := portal.LoginContext(ctx, cfg); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "[+] 认证流程完成（认证服务器返回 ok）")
	return nil
}
