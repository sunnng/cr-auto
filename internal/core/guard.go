// Package core 对应 Lua 工程的 core/ 目录：守卫、调度器、状态机与主循环。
package core

import (
	"sort"
	"time"

	"app/internal/lib/color"
	"app/internal/lib/logger"
)

const guardTag = "[Guard]"

// Trap 守卫条目：特征 + 处理函数 + 优先级。
type Trap struct {
	Name     string
	Feature  any // vision.Feature | []vision.Feature | func() bool
	Handler  func()
	Priority int
}

// Guard 弹窗守卫：比色拦截弹窗并自动处理（仅主线程调用，避免与业务并发 tap）。
type Guard struct {
	traps  map[string]Trap
	sorted []Trap
	dirty  bool
	sleep  func(ms int)
}

// NewGuard 创建空守卫。
func NewGuard() *Guard {
	return &Guard{
		traps: map[string]Trap{},
		sleep: func(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) },
	}
}

// SetSleep 注入休眠实现（测试用）。
func (g *Guard) SetSleep(fn func(ms int)) { g.sleep = fn }

// Register 注册守卫（同名覆盖），priority 越高越先扫描。
func (g *Guard) Register(name string, feature any, handler func(), priority int) {
	g.traps[name] = Trap{Name: name, Feature: feature, Handler: handler, Priority: priority}
	g.dirty = true
	logger.Info(guardTag, "注册 %s priority=%d", name, priority)
}

// Clear 清空所有守卫。
func (g *Guard) Clear() {
	if len(g.traps) > 0 {
		logger.Debug(guardTag, "清空 %d 个 trap", len(g.traps))
	}
	g.traps = map[string]Trap{}
	g.sorted = nil
	g.dirty = false
}

// TrapCount 守卫数量。
func (g *Guard) TrapCount() int { return len(g.traps) }

func (g *Guard) sortedTraps() []Trap {
	if g.dirty || g.sorted == nil {
		list := make([]Trap, 0, len(g.traps))
		for _, trap := range g.traps {
			list = append(list, trap)
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Priority != list[j].Priority {
				return list[i].Priority > list[j].Priority
			}
			return list[i].Name < list[j].Name
		})
		g.sorted = list
		g.dirty = false
	}
	return g.sorted
}

// Check 扫描并处理首个命中的守卫（按 priority 降序、同名升序）。
// 返回是否命中并已尝试处理；处理函数异常时返回 false。
func (g *Guard) Check() bool {
	for _, trap := range g.sortedTraps() {
		if ok, _ := color.Any(trap.Feature); !ok {
			continue
		}
		logger.Info(guardTag, "[命中] %s", trap.Name)
		if !safeCall(trap.Handler) {
			logger.Error(guardTag, "[处理] %s 失败", trap.Name)
			return false
		}
		logger.Info(guardTag, "[处理] %s 完成", trap.Name)
		return true
	}
	return false
}

// Sleep 分片 sleep：每片 sleep 前执行守卫扫描（长等待期间清弹窗）。
func (g *Guard) Sleep(ms, stepMs int) {
	if stepMs <= 0 {
		stepMs = 500
	}
	left := max(0, ms)
	if left >= 5000 {
		logger.Debug(guardTag, "[sleep] %dms 分片 %dms", left, stepMs)
	}
	for left > 0 {
		g.Check()
		chunk := min(left, stepMs)
		g.sleep(chunk)
		left -= chunk
	}
}

// safeCall 执行 fn 并吞掉 panic。
func safeCall(fn func()) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	fn()
	return true
}
