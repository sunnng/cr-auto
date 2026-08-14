package core

import (
	"errors"
	"time"

	"app/internal/lib/logger"
)

const smTag = "[StateMachine]"

// 状态机返回符号。
const (
	KEEP  = "__keep__"  // 保持当前状态（等待 / 多步未完成）
	RETRY = "__retry__" // 当前状态重试一次（仅显式返回时计数）
	DONE  = "__done__"  // 正常结束
)

// StateHandler 状态处理器：返回下一状态名或 KEEP/RETRY/DONE 符号；
// 返回非 nil 错误表示致命错误，终止状态机。
type StateHandler func(sm *StateMachine) (string, error)

// InitOpts 初始化选项。
type InitOpts struct {
	MaxRetry        int // 主动重试上限，默认 3
	MaxError        int // 异常重试上限，默认 3
	RetryIntervalMs int // RETRY/异常重试后的 sleep；>0 时替代本轮 interval
	TimeoutSec      int // 超时秒数，默认 1800
}

// RunOpts 运行选项。
type RunOpts struct {
	Interval int    // 正常轮询 sleep 毫秒（KEEP / 切态后）
	Guard    func() // 每轮 handler 前 + loopSleep 分片前调用（如 Guard.Check）
	Label    string // 日志前缀
}

// StateMachine 对应 Lua 工程的 core/state-machine.lua：极简状态机，实例隔离。
type StateMachine struct {
	current         string
	retries         int
	maxRetry        int
	errors          int
	maxError        int
	retryIntervalMs int
	startTime       time.Time
	timeoutSec      int64
	ticks           int

	// Ctx 任务上下文（如目标楼层等跨处理器数据）。
	Ctx any

	nowFn    func() time.Time
	sleep    func(ms int)
	runGuard func()
}

// New 创建状态机实例。
func New() *StateMachine {
	return &StateMachine{nowFn: time.Now, sleep: func(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }}
}

// SetNow 注入时钟（测试用）。
func (sm *StateMachine) SetNow(fn func() time.Time) { sm.nowFn = fn }

// SetSleep 注入休眠（测试用）。
func (sm *StateMachine) SetSleep(fn func(ms int)) { sm.sleep = fn }

// Init 以 firstState 初始化并重置计数器。
func (sm *StateMachine) Init(firstState string, opts InitOpts) {
	if opts.MaxRetry == 0 {
		opts.MaxRetry = 3
	}
	if opts.MaxError == 0 {
		opts.MaxError = 3
	}
	if opts.TimeoutSec == 0 {
		opts.TimeoutSec = 1800
	}
	sm.current = firstState
	sm.retries = 0
	sm.errors = 0
	sm.maxRetry = opts.MaxRetry
	sm.maxError = opts.MaxError
	sm.retryIntervalMs = opts.RetryIntervalMs
	sm.timeoutSec = int64(opts.TimeoutSec)
	sm.startTime = sm.nowFn()
	sm.ticks = 0
}

// To 切换到新状态并清零重试计数。
func (sm *StateMachine) To(state string) {
	sm.current = state
	sm.retries = 0
	sm.errors = 0
}

// RetryActive 主动重试计数（handler 显式返回 RETRY 时调用）。
func (sm *StateMachine) RetryActive() (bool, string) {
	sm.retries++
	if sm.retries > sm.maxRetry {
		return false, "状态 [" + sm.current + "] 主动重试超限"
	}
	return true, "主动重试"
}

// RetryError 异常重试计数（handler panic 时调用）。
func (sm *StateMachine) RetryError() (bool, string) {
	sm.errors++
	if sm.errors > sm.maxError {
		return false, "状态 [" + sm.current + "] 异常重试超限"
	}
	return true, "异常重试"
}

// Retry 向后兼容：等价 RetryActive。
func (sm *StateMachine) Retry() (bool, string) { return sm.RetryActive() }

// IsTimeout 是否超过运行时限。
func (sm *StateMachine) IsTimeout() bool {
	return sm.nowFn().Sub(sm.startTime) > time.Duration(sm.timeoutSec)*time.Second
}

// GetState 当前状态名。
func (sm *StateMachine) GetState() string { return sm.current }

// Retries 当前状态的主动重试次数（HUD 展示用，对应 Lua sm.retries）。
func (sm *StateMachine) Retries() int { return sm.retries }

// loopSleep 带守卫分片的 sleep。
func (sm *StateMachine) loopSleep(ms, interval int) {
	left := ms
	for left > 0 {
		if sm.runGuard != nil {
			safeCall(sm.runGuard)
		}
		chunk := min(left, interval)
		sm.sleep(chunk)
		left -= chunk
	}
}

// Run 运行状态机。
func (sm *StateMachine) Run(handlers map[string]StateHandler, opts RunOpts) (bool, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = 500
	}
	label := opts.Label
	if label == "" {
		label = "任务"
	}
	sm.runGuard = opts.Guard

	logger.Info(smTag, "[%s] 启动 初始=%s maxRetry=%d timeout=%ds interval=%dms",
		label, sm.current, sm.maxRetry, sm.timeoutSec, interval)

	for {
		sm.ticks++

		if sm.IsTimeout() {
			logger.Warn(smTag, "[%s] 超时 状态=%s 轮次=%d", label, sm.current, sm.ticks)
			return false, errors.New("timeout")
		}

		if opts.Guard != nil {
			safeCall(opts.Guard)
		}

		state := sm.GetState()
		handler := handlers[state]
		if handler == nil {
			logger.Warn(smTag, "[%s] 未知状态: %s", label, state)
			return false, errors.New("unknown state: " + state)
		}

		next, fatal, panicked := callHandler(handler, sm)
		retried := false

		if panicked {
			ok, msg := sm.RetryError()
			if !ok {
				logger.Warn(smTag, "[%s] %s | panic", label, msg)
				return false, errors.New(msg)
			}
			logger.Info(smTag, "[%s] [%s] %s", label, state, msg)
			retried = true
		} else if fatal != nil {
			logger.Warn(smTag, "[%s] [%s] 致命: %v", label, state, fatal)
			return false, fatal
		} else {
			switch next {
			case DONE:
				logger.Info(smTag, "[%s] 正常结束 末状态=%s 轮次=%d", label, state, sm.ticks)
				return true, nil
			case RETRY:
				ok, msg := sm.RetryActive()
				if !ok {
					logger.Warn(smTag, "[%s] %s", label, msg)
					return false, errors.New(msg)
				}
				logger.Info(smTag, "[%s] [%s] %s", label, state, msg)
				retried = true
			case KEEP, "":
				// 保持当前状态。
			default:
				logger.Info(smTag, "[%s] [%s] → %s", label, state, next)
				sm.To(next)
			}
		}

		if retried {
			if sm.retryIntervalMs > 0 {
				sm.loopSleep(sm.retryIntervalMs, interval)
			} else {
				sm.loopSleep(interval, interval)
			}
		} else {
			sm.loopSleep(interval, interval)
		}
	}
}

func callHandler(handler StateHandler, sm *StateMachine) (next string, fatal error, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			next, fatal = "", nil
		}
	}()
	next, fatal = handler(sm)
	return next, fatal, false
}
