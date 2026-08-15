// Package square 对应 Lua 工程的 game/常规_布谷鸟广场/：广场特征库、页面、会话、路由与任务。
package square

import (
	"app/internal/game/kingdom"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/touch"
)

const routeTag = "[布谷鸟广场路由]"

func waitKingdom(timeoutMs int) bool {
	return color.WaitMatch(kingdom.HomeFeature, timeoutMs, 500, 2000)
}

// KingdomToSquare 王国主城 → 广场首页（对应 SquareRoute.kingdomToSquare）。
func KingdomToSquare() bool {
	if IsCurrent() {
		return true
	}
	if !kingdom.IsKingdomHome() {
		logger.Warn(routeTag, " 不在王国主城，无法进广场")
		return false
	}
	touch.TapArea(kingdom.Home().SquareBtn, 0)
	if WaitHome(0, 0, 0) {
		logger.Info(routeTag, " 已进入广场")
		return true
	}
	logger.Warn(routeTag, " 进入广场超时")
	return false
}

// OpenLeaveDialog 打开「离开广场」弹窗（对应 SquareRoute.openLeaveDialog）。
func OpenLeaveDialog() bool {
	if IsLeaveDialog() {
		return true
	}
	if !IsCurrent() {
		if !KingdomToSquare() {
			return false
		}
	}
	TapBack(1200)
	if WaitLeaveDialog(15000, 500) {
		logger.Info(routeTag, " 已打开离开广场弹窗")
		return true
	}
	logger.Warn(routeTag, " 离开广场弹窗未出现")
	return false
}

// LeaveDialogToKingdom 经弹窗或广场返回王国主城（对应 SquareRoute.leaveDialogToKingdom）。
func LeaveDialogToKingdom(timeoutMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if kingdom.IsKingdomHome() {
		return true
	}
	if IsLeaveDialog() {
		TapReturnKingdom(1200)
	} else if IsCurrent() {
		TapBack(1000)
		if WaitLeaveDialog(8000, 500) {
			TapReturnKingdom(1200)
		}
	}
	if waitKingdom(timeoutMs) {
		logger.Info(routeTag, " 已回王国主城")
		return true
	}
	logger.Warn(routeTag, " 回王国主城超时")
	return false
}

// IsSquareContext 是否处于广场上下文（对应 SquareRoute.isSquareContext）。
func IsSquareContext() bool {
	return IsCurrent() || IsLeaveDialog()
}
