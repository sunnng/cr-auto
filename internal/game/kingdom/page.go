// Package kingdom 对应 Lua 工程的 game/通用_王国/：王国首页/活动/探险页特征库与页面。
package kingdom

import (
	"app/internal/lib/color"
	"app/internal/lib/touch"
	"app/internal/vision"
)

// IsKingdomHome 判断是否在王国首页（对应 KingdomPage.isKingdomHome）。
func IsKingdomHome() bool {
	return color.Match(home.Feature)
}

// TapEventBtn 点击王国首页“活动”按钮（对应 KingdomPage.tapEventBtn）。
func TapEventBtn() { touch.TapArea(home.EventBtn, 1200) }

// TapMineBtn 点击活动页“矿山”入口（对应 KingdomPage.tapMineBtn）。
func TapMineBtn() { touch.TapArea(event.MineBtn, 1200) }

// TapAdventureBtn 点击王国首页“探险”按钮（对应 KingdomPage.tapAdventureBtn）。
func TapAdventureBtn() { touch.TapArea(home.AdventureBtn, 1200) }

// IsAdventurePage 判断是否在探险页（对应 KingdomPage.isAdventurePage）。
func IsAdventurePage() bool {
	return color.Match(adventure.Feature)
}

// WaitAdventure 等待进入探险页（对应 KingdomPage.waitAdventure，默认 30s）。
func WaitAdventure(timeoutMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	return color.WaitMatch(adventure.Feature, timeoutMs, 500, 800)
}

// Wait 等待进入王国首页（对应 KingdomPage.wait，默认 90s）。
func Wait(timeoutMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 90000
	}
	return color.WaitMatch(home.Feature, timeoutMs, 500, 1000)
}

// HomeFeature 王国首页特征常量（供外部 color.WaitMatch 等使用，对应 KingdomPage.HOME_FEATURE）。
var HomeFeature = vision.Feature{Points: home.Feature.Points, Sim: home.Feature.Sim}
