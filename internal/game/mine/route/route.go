// Package route 对应 Lua 工程的 game/常规_未知的地底矿山/矿山_路由/：矿山导航步骤。
// 结构直译例外：Lua 允许循环 require，Go 不允许包循环；
// 涉及勘查页的 MineHomeToMineVenture / MineVentureToMineHome 随使用方
// （mine/survey）存放为 EnterMineVenture / BackToMineHome，
// 涉及开采页的 ReturnToKingdom 随使用方（mine/mining）存放。
package route

import (
	"app/internal/game/kingdom"
	"app/internal/game/mine"
)

// KingdomHomeToMineHome 王国首页 → 矿山首页（对应 Route.kingdomHomeToMineHome）。
func KingdomHomeToMineHome() bool {
	kingdom.TapEventBtn()
	kingdom.TapMineBtn()
	return mine.Wait()
}

// MineHomeToMining 矿山首页 → 开采页（对应 Route.mineHomeToMining）。
func MineHomeToMining() bool {
	mine.TapMining()
	return mine.WaitGone()
}

// MineHomeToKingdom 矿山首页 → 王国首页（对应 Route.mineHomeToKingdom）。
func MineHomeToKingdom() bool {
	mine.TapBack()
	return kingdom.Wait(0)
}
