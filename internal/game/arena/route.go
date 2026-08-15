// Package arena 对应 Lua 工程的 game/常规_王国竞技场/：竞技场特征库、页面、会话、路由与任务。
package arena

import (
	"app/internal/game/kingdom"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
)

const routeTag = "[王国竞技场.路由]"

// Enter 王国主城/冒险页 → 竞技场大厅（对应 Route.enter）。
func Enter() bool {
	if IsLobby() {
		logger.Info(routeTag, " 已在大厅，跳过导航")
		return true
	}

	if !kingdom.IsAdventurePage() {
		if !kingdom.IsKingdomHome() {
			logger.Warn(routeTag, " 不在王国主城，无法进入")
			return false
		}
		logger.Info(routeTag, " 王国主城 → 点击冒险")
		kingdom.TapAdventureBtn()
		if !kingdom.WaitAdventure(30000) {
			logger.Warn(routeTag, " 等待冒险页超时")
			return false
		}
		logger.Info(routeTag, " 已进入冒险页")
	}

	logger.Info(routeTag, " OCR 查找并点击「王国竞技场」")
	if ok, _, _ := ocr.WaitTap("王国竞技场", kingdom.Adventure().ArenaOcr, 30000, 500, 1000); !ok {
		logger.Warn(routeTag, " 未能点击王国竞技场")
		return false
	}

	if tapToLobby() {
		logger.Info(routeTag, " 已进入竞技场大厅")
		return true
	}
	logger.Warn(routeTag, " 等待竞技场大厅超时")
	return false
}

// Leave 竞技场 → 王国主城（对应 Route.leave）。
func Leave() bool {
	if kingdom.IsKingdomHome() {
		logger.Info(routeTag, " 已在王国首页，无需离开")
		return true
	}
	logger.Info(routeTag, " 离开竞技场")
	if IsLobby() {
		logger.Debug(routeTag, " 关闭竞技场大厅")
		TapClose(1200)
	}
	if kingdom.IsAdventurePage() {
		logger.Debug(routeTag, " 冒险页按返回")
		touch.PressBack(1200)
	}
	if kingdom.Wait(15000) {
		logger.Info(routeTag, " 已回到王国首页")
		return true
	}
	logger.Warn(routeTag, " 离开竞技场超时")
	return false
}
