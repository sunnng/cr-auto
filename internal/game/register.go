// Package game 对应 Lua 工程的 game/ 目录：任务构建器与业务注册。
package game

import (
	"app/internal/core"
	"app/internal/lib/logger"
)

const registerTag = "[Register]"

// RegisterAll 对应 Lua game/register.lua 的 Register.all()：清空后注入守卫与任务。
// M1（引擎底座）阶段无游戏模块，仅完成清理与占位；
// M2 起按结构直译把守卫陷阱（网络联机状态不稳定等）与调度任务（矿山/广场/交易所…）注入此处。
func RegisterAll(s *core.Scheduler, g *core.Guard) {
	s.Clear()
	g.Clear()

	// ========== 守卫注册（优先级高->低）==========
	// M2: Guard.Register("网络联机状态不稳定", popupFeature, handler, 10) ...

	// ========== 调度任务注册 ==========
	// M2: NewTask(s, uc, "矿山勘查", TaskOptions{...}) ...

	// ========== idle provider 注册 ==========
	// M2: s.AddIdleProvider("矿山勘查", ...) ...

	logger.Info(registerTag, "注入完成 | 守卫 %d 个 任务 %d 个", g.TrapCount(), s.Count())
}
