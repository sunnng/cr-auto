// Package square 对应 Lua 工程的 game/常规_布谷鸟广场/：广场特征库、页面、会话、路由与任务。
package square

import (
	"image"
	"strconv"
	"strings"

	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
)

const pageTag = "[布谷鸟广场]"

// IsCurrent 是否在广场首页（对应 SquarePage.isCurrent）。
func IsCurrent() bool { return color.Match(squareFeatures.Home.Feature) }

// WaitHome 等待广场首页出现（对应 SquarePage.waitHome，默认 30s）。
func WaitHome(timeoutMs, intervalMs, sleepMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	if sleepMs <= 0 {
		sleepMs = 1000
	}
	return color.WaitMatch(squareFeatures.Home.Feature, timeoutMs, intervalMs, sleepMs)
}

// IsLeaveDialog 是否在离开广场弹窗（对应 SquarePage.isLeaveDialog）。
func IsLeaveDialog() bool { return color.Match(squareFeatures.DialogLeave.Feature) }

// WaitLeaveDialog 等待离开广场弹窗出现（对应 SquarePage.waitLeaveDialog，默认 15s）。
func WaitLeaveDialog(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 15000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(squareFeatures.DialogLeave.Feature, timeoutMs, intervalMs, 0)
}

// TapBackBtn 点击广场首页返回按钮（不等待，对应 SquarePage.tapBackBtn）。
func TapBackBtn() { touch.TapArea(squareFeatures.Home.BackBtn, 0) }

// TapBack 点击广场首页返回按钮（对应 SquarePage.tapBack，默认 1000ms）。
func TapBack(delayMs int) {
	touch.TapArea(squareFeatures.Home.BackBtn, defaultDelay(delayMs, 1000))
}

// TapLeaveBtn 点击离开弹窗「离开」按钮（对应 SquarePage.tapLeaveBtn，默认 1200ms）。
func TapLeaveBtn(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.LeaveBtn, defaultDelay(delayMs, 1200))
}

// TapReturnKingdom 点击离开弹窗「回王国」按钮（对应 SquarePage.tapReturnKingdom）。
func TapReturnKingdom(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.LeaveBtn, defaultDelay(delayMs, 1200))
}

// TapCancelBtn 点击离开弹窗取消按钮（对应 SquarePage.tapCancelBtn）。
func TapCancelBtn(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.CancelBtn, defaultDelay(delayMs, 1200))
}

// TapCloseDialog 点击离开弹窗关闭按钮（对应 SquarePage.tapCloseDialog）。
func TapCloseDialog(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.CancelBtn, defaultDelay(delayMs, 1200))
}

// TapConfirmRewardBtn 点击确认领奖按钮（对应 SquarePage.tapConfirmRewardBtn，默认 1000ms）。
func TapConfirmRewardBtn(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.ConfirmRewardBtn, defaultDelay(delayMs, 1000))
}

// TapClaimAll 点击一次领回（对应 SquarePage.tapClaimAll）。
func TapClaimAll(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.ConfirmRewardBtn, defaultDelay(delayMs, 1000))
}

// TapUtilDialog 点击弹窗工具区域直到弹窗特征命中（对应 SquarePage.tapUtilDialog）。
func TapUtilDialog() {
	color.TapUntilMatch(image.Rect(722, 686, 886, 725), squareFeatures.DialogLeave.Feature, color.TapOpts{})
}

// TapConfirmReward 点击确认领奖（对应 SquarePage.tapConfirmReward）。
func TapConfirmReward(delayMs int) {
	touch.TapArea(squareFeatures.DialogLeave.ConfirmRewardBtn, defaultDelay(delayMs, 1000))
}

// readCount OCR 读取数值；数字识别失败时退回纯文本取数字（对应 Lua readCount）。
func readCount(rect image.Rectangle, label string) (int, bool) {
	if rect.Empty() {
		logger.Warn(pageTag, " OCR 区域未配置: %s", label)
		return 0, false
	}
	if n, ok := ocr.Number(rect); ok {
		return n, true
	}
	text := ocr.Text(rect)
	if text != "" {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, text)
		if digits != "" {
			if n, err := strconv.Atoi(digits); err == nil {
				return n, true
			}
		}
		logger.Warn(pageTag, " %s OCR 有字无数: %s", label, text)
	}
	return 0, false
}

// textIndicatesMaxed 文本是否提示已满额（对应 Lua textIndicatesMaxed）。
func textIndicatesMaxed(text string) bool {
	if text == "" {
		return false
	}
	if strings.Contains(text, "最大") {
		return true
	}
	if strings.Contains(text, "已领取") && strings.Contains(text, "奖励") {
		return true
	}
	return false
}

// GetRewardNow 目前可获得奖励（对应 SquarePage.getRewardNow）。
func GetRewardNow() (int, bool) {
	return readCount(squareFeatures.DialogLeave.RewardNowOcr, "目前可获得奖励")
}

// GetRewardTotal 累计获得奖励（对应 SquarePage.getRewardTotal）。
func GetRewardTotal() (int, bool) {
	return readCount(squareFeatures.DialogLeave.RewardTotalOcr, "累计获得奖励")
}

// IsDailyRewardsMaxed 每日奖励是否已满额（对应 SquarePage.isDailyRewardsMaxed）。
func IsDailyRewardsMaxed() bool {
	if !IsLeaveDialog() {
		return false
	}
	rect := squareFeatures.DialogLeave.IsFinishOcr
	if rect.Empty() {
		rect = squareFeatures.DialogLeave.DailyMaxOcr
	}
	if ocr.Has("最大", rect) {
		logger.Info(pageTag, " 满额标识 OCR: 最大")
		return true
	}
	scan, ok := ocr.Scan(rect, ocr.MultiMode, ocr.ReturnTypeJSON)
	if !ok {
		return false
	}
	if textIndicatesMaxed(scan.Raw) || textIndicatesMaxed(scan.Text) {
		return true
	}
	return false
}

// IsFinishOcr 是否达到最大（对应 SquarePage.isFinishOcr）。
func IsFinishOcr() bool { return IsDailyRewardsMaxed() }

// ReadRewardSum 读取目前+累计奖励之和（对应 SquarePage.readRewardSum）。
// 返回 (pending, total, sum, ok)；sum 读取失败时 ok=false。
func ReadRewardSum() (pending, total, sum int, ok bool) {
	if !IsLeaveDialog() {
		logger.Warn(pageTag, " 不在离开广场弹窗，无法 OCR")
		return 0, 0, 0, false
	}
	pending, okP := GetRewardNow()
	total, okT := GetRewardTotal()
	if !okP || !okT {
		logger.Warn(pageTag, " 奖励 OCR 失败 目前=%d 累计=%d", pending, total)
		return pending, total, 0, false
	}
	sum = pending + total
	logger.Info(pageTag, " 奖励 可获得=%d 累计=%d 总计=%d", pending, total, sum)
	return pending, total, sum, true
}

// ReadJelliesSum 读取果冻奖励和（对应 SquarePage.readJelliesSum）。
func ReadJelliesSum() (pending, total, sum int, ok bool) {
	return ReadRewardSum()
}

func defaultDelay(delayMs, fallback int) int {
	if delayMs <= 0 {
		return fallback
	}
	return delayMs
}
