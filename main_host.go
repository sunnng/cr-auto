package main

import (
	"context"
	"sync"

	"app/internal/core"
	"app/internal/game"
	"app/internal/lib/logger"
	"app/internal/ui"
)

// Host 面板命令与平台生命周期的宿主接线：把 ui.Command 与 apkctl 事件
// 翻译成对引擎（core.Runtime）的控制。设备适配（截图/触控/日志/状态）在
// main 启动时注入；Host 本身不依赖 AutoGo，可桌面测试。
type Host struct {
	panel *ui.Panel

	mu      sync.Mutex
	rt      *core.Runtime
	cancel  context.CancelFunc
	running bool
}

func NewHost(panel *ui.Panel) *Host {
	return &Host{panel: panel}
}

// Handle 消费面板命令。
func (h *Host) Handle(command ui.Command) {
	switch command.Type {
	case ui.CommandStart:
		h.start()
	case ui.CommandPause:
		h.pause()
	case ui.CommandResume:
		h.resume()
	case ui.CommandStop:
		h.stop()
	case ui.CommandSave:
		logger.Info("[Host]", "保存设置（M1 引擎无任务配置，忽略）")
	case ui.CommandDiagnostic:
		logger.Info("[Host]", "识别诊断快照（M2 接入识别页）")
	case ui.CommandInspect:
		logger.Info("[Host]", "手动识别检查（M2 接入识别页）")
	default:
		logger.Warn("[Host]", "未知命令: %s", command.Type)
	}
}

// start 启动引擎：重建运行时并注入业务注册。
func (h *Host) start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt := core.NewRuntime(core.RuntimeOptions{})
	rt.Register = func() { game.RegisterAll(rt.Scheduler, rt.Guard) }

	h.rt = rt
	h.cancel = cancel
	h.running = true

	go func() {
		_ = rt.Run(ctx)
		h.mu.Lock()
		h.running = false
		h.mu.Unlock()
		h.panel.PublishPhase("idle", "stopped", "引擎已停止")
	}()
	h.panel.PublishPhase("running", "running", "引擎已启动（M1 引擎底座，无游戏任务）")
}

// pause 暂停引擎（轮间生效）。
func (h *Host) pause() {
	h.mu.Lock()
	rt := h.rt
	h.mu.Unlock()
	if rt == nil {
		return
	}
	rt.Pause()
	h.panel.PublishPhase("paused", "paused", "引擎已暂停")
}

// resume 恢复引擎。
func (h *Host) resume() {
	h.mu.Lock()
	rt := h.rt
	h.mu.Unlock()
	if rt == nil {
		return
	}
	rt.Resume()
	h.panel.PublishPhase("running", "running", "引擎已恢复")
}

// stop 停止引擎；返回是否真的停止（无引擎在跑时为 false）。
func (h *Host) stop() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running {
		return false
	}
	if h.cancel != nil {
		h.cancel()
	}
	return true
}
