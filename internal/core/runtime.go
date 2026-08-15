// Package core 对应 Lua 工程的 core/ 目录：守卫、调度器、状态机与主循环。
package core

import (
	"context"
	"errors"
	"strings"
	"sync"

	"app/internal/config"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
)

const runtimeTag = "[Runtime]"

// RuntimeOptions 主循环构造参数。
type RuntimeOptions struct {
	Scheduler   *Scheduler                    // nil 时自建
	Guard       *Guard                        // nil 时自建
	Register    func()                        // 业务注入点（game.Register.All）
	RoundHook   func(round int, hasWork bool) // 每轮调度结束后回调（运行模式/安全策略用）
	StopOnError bool
}

// Runtime 对应 Lua 工程的 core/runtime.lua：脚本永久运行引擎。
// 每轮：先扫守卫，再跑一轮调度；无任务时按空闲提供者结果等待。
type Runtime struct {
	Scheduler   *Scheduler
	Guard       *Guard
	Register    func()
	RoundHook   func(round int, hasWork bool)
	StopOnError bool

	GuardIntervalMS int
	IdleDelayMS     int
	StepDelayMS     int

	done    chan struct{}
	pauseMu sync.Mutex
	paused  bool
	resume  chan struct{}
}

// NewRuntime 创建主循环实例；节奏参数缺省取 config.Static.Runtime。
func NewRuntime(opts RuntimeOptions) *Runtime {
	rt := &Runtime{
		Scheduler:       opts.Scheduler,
		Guard:           opts.Guard,
		Register:        opts.Register,
		RoundHook:       opts.RoundHook,
		StopOnError:     opts.StopOnError,
		GuardIntervalMS: config.Static.Runtime.GuardIntervalMS,
		IdleDelayMS:     config.Static.Runtime.IdleDelayMS,
		StepDelayMS:     config.Static.Runtime.StepDelayMS,
		done:            make(chan struct{}),
	}
	if rt.Scheduler == nil {
		rt.Scheduler = NewScheduler()
	}
	if rt.Guard == nil {
		rt.Guard = NewGuard()
	}
	return rt
}

// Done 在 Run 返回后关闭。
func (rt *Runtime) Done() <-chan struct{} { return rt.done }

// Pause 暂停主循环（在轮间生效）。
func (rt *Runtime) Pause() {
	rt.pauseMu.Lock()
	defer rt.pauseMu.Unlock()
	if rt.paused {
		return
	}
	rt.paused = true
	rt.resume = make(chan struct{})
}

// Resume 恢复主循环。
func (rt *Runtime) Resume() {
	rt.pauseMu.Lock()
	defer rt.pauseMu.Unlock()
	if !rt.paused {
		return
	}
	rt.paused = false
	close(rt.resume)
}

// Run 永久运行：清理 → 注册 → 主循环调度 + 守卫；ctx 取消或调度异常时返回。
func (rt *Runtime) Run(ctx context.Context) error {
	defer close(rt.done)

	rt.Scheduler.Clear()
	rt.Guard.Clear()
	status.Set(status.PhaseRun, "运行中")

	// 对应 Lua runtime.lua 的 Color.setGuardHook(Guard.check)：
	// wait/sleep 分片轮询内由 lib/color 的 TickGuard 扫描弹窗。
	color.SetGuardHook(func() { rt.Guard.Check() })
	defer color.SetGuardHook(nil)

	if rt.Register != nil {
		rt.Register()
	}

	round := 0
	for {
		round++
		logger.Debug(runtimeTag, "[轮次] #%d 开始", round)

		if ctx.Err() != nil {
			return nil
		}
		if !rt.waitWhilePaused(ctx) {
			return nil
		}

		rt.Guard.Check()
		hasWork, ok := rt.Scheduler.Run(rt.StopOnError)
		if !ok {
			logger.Warn(runtimeTag, "[轮次] #%d 调度异常终止", round)
			return errors.New("调度异常终止")
		}
		if rt.RoundHook != nil {
			rt.RoundHook(round, hasWork)
			if ctx.Err() != nil {
				return nil
			}
		}

		if !hasWork {
			if !rt.idleWait(ctx) {
				return nil
			}
		} else {
			logger.Debug(runtimeTag, "[step] 轮间等待 %ds", rt.StepDelayMS/1000)
			rt.ctxSleep(ctx, rt.StepDelayMS)
		}
	}
}

// ctxSleep 分片 sleep：每片间隙检查 ctx 与守卫，保证取消/暂停可及时生效。
func (rt *Runtime) ctxSleep(ctx context.Context, ms int) {
	left := max(0, ms)
	for left > 0 {
		if ctx.Err() != nil {
			return
		}
		chunk := min(left, rt.GuardIntervalMS)
		rt.Guard.Sleep(chunk, rt.GuardIntervalMS)
		left -= chunk
	}
}

// idleWait 按空闲提供者结果等待；ctx 取消时返回 false。
func (rt *Runtime) idleWait(ctx context.Context) bool {
	remain, hud := rt.calcIdleWait()
	if remain > 0 {
		idleSec := max(1, rt.IdleDelayMS/1000)
		logger.Info(runtimeTag, "[idle] 等待 剩余%ds tick %ds | %s", remain, idleSec, hud)
		for i := 0; i < idleSec; i++ {
			if ctx.Err() != nil {
				return false
			}
			remain, hud = rt.calcIdleWait()
			if remain <= 0 {
				logger.Info(runtimeTag, "[idle] 等待已到期")
				break
			}
			status.SetWait(hud)
			rt.ctxSleep(ctx, 1000)
		}
		return ctx.Err() == nil
	}
	status.SetIdle()
	logger.Info(runtimeTag, "[idle] 无任务 挂机 %ds", rt.IdleDelayMS/1000)
	rt.ctxSleep(ctx, rt.IdleDelayMS)
	return ctx.Err() == nil
}

// calcIdleWait 计算所有空闲提供者的最大等待秒数与等待提示文本。
func (rt *Runtime) calcIdleWait() (waitRemain int, hudText string) {
	var parts []string
	for name, provider := range rt.Scheduler.GetIdleProviders() {
		remain, label := safeIdleProvider(name, provider)
		if remain > 0 {
			if remain > waitRemain {
				waitRemain = remain
			}
			if label != "" {
				parts = append(parts, label)
			}
		}
	}
	return waitRemain, strings.Join(parts, " · ")
}

// safeIdleProvider 吞掉提供者 panic。
func safeIdleProvider(name string, provider IdleProvider) (remain int, label string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warn(runtimeTag, "idle provider [%s] 异常", name)
		}
	}()
	return provider()
}

// waitWhilePaused 暂停期间阻塞；返回 false 表示 ctx 已取消。
func (rt *Runtime) waitWhilePaused(ctx context.Context) bool {
	rt.pauseMu.Lock()
	if !rt.paused {
		rt.pauseMu.Unlock()
		return true
	}
	resume := rt.resume
	rt.pauseMu.Unlock()

	select {
	case <-resume:
		return true
	case <-ctx.Done():
		return false
	}
}
