// Package survey 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山勘查/：勘查页面与会话。
package survey

import (
	"app/internal/game/mine"
	"app/internal/lib/color"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
	"app/internal/vision"
)

var features = mine.MineVenture()

// IsMineVentureDomain 判断是否在矿山勘查域（对应 mineVenturePage.isMineVentureDomain）。
func IsMineVentureDomain() bool {
	ok, _ := color.MatchAny([]vision.Feature{
		features.Setup.Feature,
		features.Ready.Feature,
		features.Running.Feature,
		features.Settle.Feature,
	})
	return ok
}

// WaitMineVentureDomain 等待进入矿山勘查域（对应 mineVenturePage.waitMineVentureDomain，60s）。
func WaitMineVentureDomain() bool {
	ok, _ := color.Wait([]vision.Feature{
		features.Setup.Feature,
		features.Ready.Feature,
		features.Running.Feature,
		features.Settle.Feature,
	}, 60000, 500)
	return ok
}

// IsSetup 判断是否在勘查准备页（对应 mineVenturePage.isSetup）。
func IsSetup() bool { return color.Match(features.Setup.Feature) }

// IsReady 判断是否在勘查就绪页（对应 mineVenturePage.isReady）。
func IsReady() bool { return color.Match(features.Ready.Feature) }

// IsRunning 判断是否在勘查运行页（对应 mineVenturePage.isRunning）。
func IsRunning() bool { return color.Match(features.Running.Feature) }

// TapStopBtn 点击停止勘查按钮（对应 mineVenturePage.tapStopBtn）。
func TapStopBtn() { touch.TapArea(features.Running.StopBtn, 500) }

// Setup 启动勘查流程：自动选择 → 开始 → 两段确认 → 运行（对应 mineVenturePage.setup）。
func Setup() bool {
	touch.TapArea(features.Setup.AutoSelectBtn, 500)
	if !color.WaitMatch(features.Ready.Feature, 30000, 500, 1000) {
		return false
	}
	touch.TapArea(features.Ready.StartBtn, 500)
	if !color.WaitMatch(features.DialogInfo.Feature, 10000, 500, 1000) {
		return false
	}
	touch.TapArea(features.DialogInfo.ConfirmBtn, 500)
	if !color.WaitMatch(features.DialogConfirmCookie.Feature, 10000, 500, 1000) {
		return false
	}
	touch.TapArea(features.DialogConfirmCookie.ConfirmBtn, 500)
	if !color.WaitMatch(features.Running.Feature, 15000, 500, 1000) {
		return false
	}
	return true
}

// StopVenture 停止勘查并结算（对应 mineVenturePage.stopVenture）。
func StopVenture() bool {
	TapStopBtn()
	if !color.WaitMatch(features.DialogStop.Feature, 10000, 500, 1000) {
		return false
	}
	touch.TapArea(features.DialogStop.ConfirmStopBtn, 500)
	color.TapUntilMatch(
		features.Settle.FinishBtn,
		features.Setup.Feature,
		color.TapOpts{TimeoutMs: 20000, TapDelayMs: 800, IntervalMs: 500, SleepMs: 500},
	)
	return true
}

// GetCurrentFloor 读取当前层数（对应 mineVenturePage.getCurrentFloor）。
func GetCurrentFloor() (int, bool) {
	return ocr.Number(features.Running.FloorOcr)
}

// TapBackBtn 点击返回按钮（对应 mineVenturePage.tapBackBtn）。
func TapBackBtn() { touch.TapArea(features.BackBtn, 1000) }

// EnterMineVenture 矿山首页 → 勘查域（对应 Route.mineHomeToMineVenture；
// 因 Go 包循环限制随使用方存放）。
func EnterMineVenture() bool {
	mine.TapVenture()
	return WaitMineVentureDomain()
}

// BackToMineHome 勘查域 → 矿山首页（对应 Route.mineVentureToMineHome；
// 因 Go 包循环限制随使用方存放）。
func BackToMineHome() bool {
	TapBackBtn()
	return mine.Wait()
}
