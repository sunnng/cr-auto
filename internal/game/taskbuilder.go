// Package game 对应 Lua 工程的 game/ 目录：任务构建器与业务注册。
package game

import (
	"errors"
	"fmt"

	"app/internal/core"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/userconfig"
)

const taskBuilderTag = "[TaskBuilder]"

// TaskOptions 标准任务构建参数，对应 Lua task-builder.lua 的 opts。
type TaskOptions struct {
	// ConfigKey UserConfig 段落名，检查 cfg.Enabled == true。
	ConfigKey string
	// CheckEnabled 自定义开关检查（优先级高于 ConfigKey）。
	CheckEnabled func() bool
	// CanResume 页面恢复：返回 true 时跳过 CheckReady。
	CanResume func() bool
	// CheckReady 业务就绪检查：返回 (canRun, remainSec)。
	CheckReady func() (canRun bool, remainSec int)
	// WaitHud 等待时 HUD 文本（remain -> string）。
	WaitHud func(remain int) string
	// Precondition 额外前置条件，返回 false 则跳过。
	Precondition func() bool
	// OnPreconditionFail precondition 失败时回调。
	OnPreconditionFail func()
	// OnNotReady checkReady 返回 false 时回调（remain 作为参数）。
	OnNotReady func(remain int)
	// LeaveSquare 执行前尝试离开广场（M2 广场模块接入后由 register 注入）。
	LeaveSquare func() bool
	// Action 任务执行体。
	Action func() error
}

// NewTask 创建并注册一个标准任务（封装开关、就绪、让渡、离开广场、日志）。
func NewTask(s *core.Scheduler, uc *userconfig.UserConfig, name string, opts TaskOptions) {
	condition := func() bool {
		// 1. 开关检查。
		if opts.CheckEnabled != nil {
			if !opts.CheckEnabled() {
				return false
			}
		} else if opts.ConfigKey != "" {
			var cfg struct{ Enabled bool }
			if err := uc.Get(opts.ConfigKey, &cfg); err != nil {
				logger.Warn(taskBuilderTag, "%s 读取配置失败: %v", name, err)
				return false
			}
			if !cfg.Enabled {
				return false
			}
		}

		// 2. 额外前置条件。
		if opts.Precondition != nil && !opts.Precondition() {
			logger.Info(taskBuilderTag, "%s 前置条件未通过，跳过", name)
			if opts.OnPreconditionFail != nil {
				opts.OnPreconditionFail()
			}
			return false
		}

		// 3. 页面恢复优先。
		if opts.CanResume != nil && opts.CanResume() {
			return true
		}

		// 4. 业务就绪检查。
		if opts.CheckReady != nil {
			canRun, remain := opts.CheckReady()
			if !canRun {
				if opts.WaitHud != nil && remain > 0 {
					text := opts.WaitHud(remain)
					status.SetTask(name, text)
					logger.Info(taskBuilderTag, "%s 等待中: %s", name, text)
				}
				if opts.OnNotReady != nil {
					opts.OnNotReady(remain)
				}
				return false
			}
		}

		return true
	}

	action := func() error {
		if opts.LeaveSquare != nil && !opts.LeaveSquare() {
			logger.Warn(taskBuilderTag, "%s 因离开广场失败跳过", name)
			return core.ErrSkip
		}
		logger.Info(taskBuilderTag, "开始 %s", name)
		// Lua 的 task-builder 用 pcall 吞掉动作 panic 并当作“结束 false”；
		// 这里保留错误交给调度器（stopOnError 时终止），行为更安全。
		err := runAction(opts.Action)
		if err != nil {
			if errors.Is(err, core.ErrSkip) {
				logger.Warn(taskBuilderTag, "%s 结束 false", name)
			} else {
				logger.Error(taskBuilderTag, "%s 异常: %v", name, err)
			}
			return err
		}
		logger.Info(taskBuilderTag, "%s 完成", name)
		return nil
	}

	s.Add(name, condition, action)
}

// runAction 执行动作并把 panic 转为错误。
func runAction(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("taskbuilder: panic: %v", r)
		}
	}()
	return fn()
}
