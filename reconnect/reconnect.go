// Package reconnect checks the portal state and logs in after a confirmed disconnect.
package reconnect

import (
	"context"
	"fmt"
	"time"

	"srun-auth/config"
	"srun-auth/portal"
)

// Run checks immediately, then waits interval after each completed check.
// Cancellation stops normally. A permanent account rejection pauses authentication
// until the process is stopped, so a service manager will not restart a bad login.
func Run(ctx context.Context, cfg *config.Config, interval time.Duration, logf func(string, ...any)) error {
	r := runner{
		cfg:       cfg,
		logf:      logf,
		status:    portal.StatusContext,
		login:     portal.LoginContext,
		permanent: portal.IsPermanent,
	}
	return r.run(ctx, interval)
}

type runner struct {
	cfg       *config.Config
	logf      func(string, ...any)
	status    func(context.Context, *config.Config) (portal.State, error)
	login     func(context.Context, *config.Config) error
	permanent func(error) bool

	stateKnown bool
	state      portal.State
	lastError  string
	paused     bool
}

func (r *runner) run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("检查间隔必须大于 0")
	}
	for ctx.Err() == nil {
		r.step(ctx)
		if r.paused {
			<-ctx.Done()
			return nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
	return nil
}

func (r *runner) step(ctx context.Context) {
	if ctx.Err() != nil || r.paused {
		return
	}
	state, err := r.status(ctx, r.cfg)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		r.reportState(portal.Unknown)
		r.reportError(fmt.Sprintf("查询认证状态失败：%v", err))
		return
	}
	r.reportState(state)
	if state != portal.Offline {
		return
	}
	if ctx.Err() != nil {
		return
	}
	if err := r.login(ctx, r.cfg); err != nil {
		if ctx.Err() != nil {
			return
		}
		if r.permanent(err) {
			r.paused = true
			r.reportError(fmt.Sprintf("账号认证被拒绝，已暂停重连；修改配置后重启程序：%v", err))
		} else {
			r.reportError(fmt.Sprintf("重连失败，等待下次检查：%v", err))
		}
		return
	}
	if ctx.Err() != nil {
		return
	}
	state, err = r.status(ctx, r.cfg)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		r.reportState(portal.Unknown)
		r.reportError(fmt.Sprintf("认证请求已接受，但查询在线状态失败：%v", err))
		return
	}
	r.reportState(state)
	if state != portal.Online {
		r.reportError("认证请求已接受，但尚未确认在线；等待下次检查")
	}
}

func (r *runner) reportState(state portal.State) {
	if state == portal.Online {
		r.lastError = ""
	}
	if r.stateKnown && r.state == state {
		return
	}
	r.stateKnown, r.state = true, state
	switch state {
	case portal.Online:
		r.logf("认证状态：在线")
	case portal.Offline:
		r.logf("认证状态：离线")
	default:
		r.logf("认证状态：未知，等待下次检查")
	}
}

func (r *runner) reportError(message string) {
	if message == r.lastError {
		return
	}
	r.lastError = message
	r.logf("%s", message)
}
