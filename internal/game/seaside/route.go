// Package seaside 对应 Lua 工程的 game/常规_海滩交易所/：交易所坐标库、页面、会话、路由与任务。
package seaside

import (
	"app/internal/game/kingdom"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/touch"
)

const routeTag = "[海滩交易所.路由]"

func isEventPage() bool {
	return color.Match(kingdom.Event().Feature)
}

func waitEvent(timeoutMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	return color.WaitMatch(kingdom.Event().Feature, timeoutMs, 500, 800)
}

// Enter 王国主城/活动页 → 海滩交易所（对应 Route.enter）。
func Enter() bool {
	if IsCurrent() {
		return true
	}
	if !isEventPage() {
		if !kingdom.IsKingdomHome() {
			logger.Warn(routeTag, " 不在王国主城/活动页，无法进入")
			return false
		}
		kingdom.TapEventBtn()
		if !waitEvent(30000) {
			logger.Warn(routeTag, " 等待王国活动页超时")
			return false
		}
	}
	touch.TapArea(kingdom.Event().SeasideMarketBtn, 1200)
	if WaitCurrent(30000, 500) {
		logger.Info(routeTag, " 已进入海滩交易所")
		return true
	}
	logger.Warn(routeTag, " 进入海滩交易所超时")
	return false
}

// Leave 海滩交易所 → 王国主城（对应 Route.leave）。
func Leave() bool {
	if kingdom.IsKingdomHome() {
		return true
	}
	if IsCurrent() {
		TapClose(1200)
	}
	if kingdom.Wait(15000) {
		logger.Info(routeTag, " 已回王国主城")
		return true
	}
	if isEventPage() {
		touch.PressBack(1200)
		if kingdom.Wait(15000) {
			logger.Info(routeTag, " 已从活动页回王国主城")
			return true
		}
	}
	logger.Warn(routeTag, " 回王国主城失败")
	return false
}

// IsMarketContext 是否处于交易所上下文（对应 Route.isMarketContext）。
func IsMarketContext() bool { return IsCurrent() }
