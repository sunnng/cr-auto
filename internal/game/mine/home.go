// Package mine 对应 Lua 工程的 game/常规_未知的地底矿山/：矿山特征库与矿山首页页面。
package mine

import (
	"app/internal/lib/color"
	"app/internal/lib/touch"
)

// IsCurrent 判断是否在矿山首页（对应 MineHomePage.isCurrent）。
func IsCurrent() bool {
	return color.Match(mineHome.Feature)
}

// HasMiningCompletedTask 矿山首页是否存在已完成的开采任务（对应 MineHomePage.hasMiningCompletedTask）。
func HasMiningCompletedTask() bool {
	return color.Match(mineHome.HasMiningCompletedTaskFeature)
}

// Wait 等待进入矿山首页（对应 MineHomePage.wait，60s）。
func Wait() bool {
	return color.WaitMatch(mineHome.Feature, 60000, 500, 1000)
}

// WaitGone 等待离开矿山首页（对应 MineHomePage.waitGone，30s）。
func WaitGone() bool {
	return color.WaitGone(mineHome.Feature, 30000, 500)
}

// TapVenture 点击勘查入口（对应 MineHomePage.tapVenture）。
func TapVenture() { touch.TapArea(mineHome.VentureBtn, 1000) }

// TapMining 点击开采入口（对应 MineHomePage.tapMining）。
func TapMining() { touch.TapArea(mineHome.MiningBtn, 1000) }

// TapBattleBtn 点击战斗入口（对应 MineHomePage.tapBattleBtn）。
func TapBattleBtn() { touch.TapArea(mineHome.BattleBtn, 1000) }

// TapBack 点击返回（对应 MineHomePage.tapBack）。
func TapBack() { touch.TapArea(mineHome.BackBtn, 1000) }
