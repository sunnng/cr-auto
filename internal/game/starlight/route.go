// Package starlight 对应 Lua 工程的 game/常规_梦幻繁星岛/：繁星岛坐标库、页面、会话、路由与任务。
package starlight

import (
	"app/internal/game/kingdom"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/touch"
)

const routeTag = "[梦幻繁星岛.路由]"

// IsStarlightHome 是否在繁星岛首页（对应 Route.isStarlightHome）。
func IsStarlightHome() bool { return IsHomePage() }

// KingdomToStarlightHome 王国首页 → 繁星岛首页（对应 Route.kingdomToStarlightHome）。
func KingdomToStarlightHome() bool {
	if IsHomePage() {
		return true
	}

	if !kingdom.IsKingdomHome() {
		logger.Warn(routeTag, " 当前不在王国首页，无法导航")
		return false
	}

	// 1. 点击王国首页事件按钮。
	if kingdom.Home().EventBtn.Empty() {
		logger.Warn(routeTag, " 王国 eventBtn 未配置")
		return false
	}
	touch.TapArea(kingdom.Home().EventBtn, 1200)

	// 2. 等待事件页。
	if !color.WaitMatch(kingdom.Event().Feature, 10000, 500, 0) {
		logger.Warn(routeTag, " 等待事件页超时")
		return false
	}

	// 3. 点击梦幻繁星岛按钮。
	if kingdom.Event().StarlightBtn.Empty() {
		logger.Warn(routeTag, " 梦幻繁星岛_按钮 未配置")
		return false
	}
	touch.TapArea(kingdom.Event().StarlightBtn, 1500)

	// 4. 等待繁星岛首页。
	return WaitHomePage(30000, 500)
}

// EnsureStarlightHome 确保位于繁星岛首页（对应 Route.ensureStarlightHome）。
func EnsureStarlightHome() bool {
	if IsHomePage() {
		return true
	}
	return KingdomToStarlightHome()
}
