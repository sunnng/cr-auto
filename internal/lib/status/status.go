// Package status 对应 Lua 工程的 lib/status-hud.lua 的调度侧接口：主循环/调度器/任务构建器
// 把运行阶段与等待信息发布到宿主（设备端由 main 接到控制面板，桌面测试注入记录器）。
package status

import "sync"

// Phase 运行阶段。
type Phase string

const (
	PhaseRun  Phase = "run"
	PhaseIdle Phase = "idle"
	PhaseWait Phase = "wait"
	PhaseTask Phase = "task"
)

// Update 一次状态发布。
type Update struct {
	Phase Phase
	Task  string // PhaseTask 时的任务名
	Text  string // 等待/任务提示文本
}

// Sink 状态输出端。
type Sink func(Update)

var (
	mu   sync.RWMutex
	sink Sink
)

// SetSink 注入输出端并返回旧输出端；nil 表示丢弃。
func SetSink(fn Sink) Sink {
	mu.Lock()
	defer mu.Unlock()
	prev := sink
	sink = fn
	return prev
}

func publish(update Update) {
	mu.RLock()
	fn := sink
	mu.RUnlock()
	if fn != nil {
		fn(update)
	}
}

// Set 设置主阶段文本（如运行中）。
func Set(phase Phase, text string) { publish(Update{Phase: phase, Text: text}) }

// SetTask 设置当前任务及提示文本。
func SetTask(name, text string) { publish(Update{Phase: PhaseTask, Task: name, Text: text}) }

// SetWait 设置等待提示文本（空闲等待阶段）。
func SetWait(text string) { publish(Update{Phase: PhaseWait, Text: text}) }

// SetIdle 设置空闲挂机状态。
func SetIdle() { publish(Update{Phase: PhaseIdle}) }
