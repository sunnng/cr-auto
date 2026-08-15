// Package starlight 对应 Lua 工程的 game/常规_梦幻繁星岛/：繁星岛坐标库、页面、会话、路由与任务。
package starlight

import (
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/touch"
)

const pageTag = "[梦幻繁星岛.页面]"

// ========== 首页 ==========

// IsHomePage 是否在繁星岛首页（对应 Page.isHomePage）。
func IsHomePage() bool { return color.Match(starlightFeatures.Home.Feature) }

// WaitHomePage 等待首页出现（对应 Page.waitHomePage，默认 30s）。
func WaitHomePage(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(starlightFeatures.Home.Feature, timeoutMs, intervalMs, 0)
}

// TapSailingManual 点击航海手册按钮（对应 Page.tapSailingManual，默认 1000ms）。
func TapSailingManual(delayMs int) bool {
	btn := starlightFeatures.Home.SailingManualBtn
	if btn.Empty() {
		logger.Warn(pageTag, " 航海手册_按钮 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 1000))
	return true
}

// TapTaskBtn 点击任务按钮（对应 Page.tapTaskBtn，默认 1000ms）。
func TapTaskBtn(delayMs int) bool {
	btn := starlightFeatures.Home.TaskBtn
	if btn.Empty() {
		logger.Warn(pageTag, " taskBtn 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 1000))
	return true
}

// TapBackToKingdom 点击首页返回王国按钮（对应 Page.tapBackToKingdom，默认 1200ms）。
func TapBackToKingdom(delayMs int) bool {
	btn := starlightFeatures.Home.BackBtn
	if btn.Empty() {
		logger.Warn(pageTag, " home.backBtn 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 1200))
	return true
}

// ========== 航海手册页 ==========

// IsManualPage 是否在航海手册页（对应 Page.isManualPage）。
func IsManualPage() bool { return color.Match(starlightFeatures.Manual.Feature) }

// WaitManualPage 等待航海手册页出现（对应 Page.waitManualPage，默认 10s）。
func WaitManualPage(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(starlightFeatures.Manual.Feature, timeoutMs, intervalMs, 0)
}

// TapLoginIsland 点击登陆回忆小岛按钮（对应 Page.tapLoginIsland，默认 1000ms）。
func TapLoginIsland(delayMs int) bool {
	btn := starlightFeatures.Manual.LoginIslandBtn
	if btn.Empty() {
		logger.Warn(pageTag, " 登陆回忆小岛_按钮 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 1000))
	return true
}

// ========== 纯香草小岛页 ==========

// IsVanillaIslandPage 是否在纯香草小岛页（对应 Page.isVanillaIslandPage）。
func IsVanillaIslandPage() bool { return color.Match(starlightFeatures.Vanilla.Feature) }

// WaitVanillaIslandPage 等待纯香草小岛页出现（对应 Page.waitVanillaIslandPage，默认 10s）。
func WaitVanillaIslandPage(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(starlightFeatures.Vanilla.Feature, timeoutMs, intervalMs, 0)
}

// TapBackFromVanilla 从纯香草小岛返回（对应 Page.tapBackFromVanilla，默认 1200ms）。
func TapBackFromVanilla(delayMs int) bool {
	btn := starlightFeatures.Vanilla.BackBtn
	if btn.Empty() {
		logger.Warn(pageTag, " 纯香草小岛.backBtn 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 1200))
	return true
}

// ========== 任务页 ==========

// IsTaskPage 是否在任务页（对应 Page.isTaskPage）。
func IsTaskPage() bool { return color.Match(starlightFeatures.Task.Feature) }

// WaitTaskPage 等待任务页出现（对应 Page.waitTaskPage，默认 10s）。
func WaitTaskPage(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(starlightFeatures.Task.Feature, timeoutMs, intervalMs, 0)
}

// TapBackFromTask 从任务页返回（对应 Page.tapBackFromTask，默认 1200ms）。
func TapBackFromTask(delayMs int) bool {
	btn := starlightFeatures.Task.BackBtn
	if btn.Empty() {
		logger.Warn(pageTag, " 任务.backBtn 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 1200))
	return true
}

// FindClaimableBtn 查找可领奖按钮坐标（对应 Page.findClaimableBtn）。
func FindClaimableBtn() (x, y int, ok bool) {
	return color.Find(starlightFeatures.Task.ClaimableBtn)
}

// TapClaimableBtn 点击可领奖按钮（对应 Page.tapClaimableBtn，默认 800ms）。
func TapClaimableBtn(x, y, delayMs int) {
	touch.TapR(x, y, defaultDelay(delayMs, 800))
}

// DismissRewardPopupIfNeeded 领奖后若出现奖励弹窗则点击中央空白处关闭（对应 Page.dismissRewardPopupIfNeeded）。
func DismissRewardPopupIfNeeded() {
	if IsTaskPage() {
		return
	}
	touch.TapR(800, 450, 500)
	color.WaitMatch(starlightFeatures.Task.Feature, 5000, 300, 0)
}

func defaultDelay(delayMs, fallback int) int {
	if delayMs <= 0 {
		return fallback
	}
	return delayMs
}
