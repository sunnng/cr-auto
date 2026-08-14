// Package battle 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山战斗/：战斗页面与会话。
package battle

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"

	"app/internal/game/mine"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
	"app/internal/vision"
)

const pageTag = "[矿山战斗.页面]"

var features = mine.Battle()

var soulStoneCategories = mine.SoulStoneCategories()

func hasFeature(f vision.Feature) bool { return f.Points != "" }

// IsBattlePage 判断是否在矿山战斗页（对应 BattlePage.isBattlePage）。
func IsBattlePage() bool {
	return hasFeature(features.Feature) && color.Match(features.Feature)
}

// WaitBattlePage 等待矿山战斗页出现（对应 BattlePage.waitBattlePage，默认 30s）。
func WaitBattlePage(timeoutMs, intervalMs int) bool {
	if !hasFeature(features.Feature) {
		return false
	}
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(features.Feature, timeoutMs, intervalMs, 1000)
}

// TapBackBtn 点击战斗页返回按钮（对应 BattlePage.tapBackBtn）。
func TapBackBtn() { touch.TapArea(features.BackBtn, 1000) }

// FindQuickBattleButton 查找快转按钮（对应 BattlePage.findQuickBattleButton）。
func FindQuickBattleButton() (x, y int, ok bool) {
	if !hasFindDef(features.QuickBattleBtn) {
		return 0, 0, false
	}
	return color.Find(features.QuickBattleBtn)
}

func hasFindDef(def vision.FindDef) bool { return def.FirstColor != "" }

// TapQuickBattleButton 点击指定坐标的快转按钮（对应 BattlePage.tapQuickBattleButton）。
func TapQuickBattleButton(x, y int) { touch.TapR(x, y, 1000) }

// WaitQuickBattleDialog 等待快转弹窗出现（对应 BattlePage.waitQuickBattleDialog，默认 10s）。
func WaitQuickBattleDialog(timeoutMs, intervalMs int) bool {
	if !hasFeature(features.QuickDialog.Feature) {
		return false
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(features.QuickDialog.Feature, timeoutMs, intervalMs, 500)
}

// WaitQuickBattleDialogGone 等待快转弹窗消失（对应 BattlePage.waitQuickBattleDialogGone，默认 10s）。
func WaitQuickBattleDialogGone(timeoutMs, intervalMs int) bool {
	if !hasFeature(features.QuickDialog.Feature) {
		return true
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitGone(features.QuickDialog.Feature, timeoutMs, intervalMs)
}

var clockCountRe = regexp.MustCompile(`(\d+)/(\d+)`)

// ReadClockCount 读取快转发条数量（使用/持有；对应 BattlePage.readClockCount）。
// 返回 (used, owned, raw, ok)；ok=false 表示无法解析（对应 Lua nil 返回值）。
func ReadClockCount() (int, int, string, bool) {
	if !hasRect(features.QuickDialog.CountOcr) {
		logger.Warn(pageTag, " 快转发条数量_Ocr 未配置")
		return 0, 0, "", false
	}
	text := ocr.Text(features.QuickDialog.CountOcr)
	if text == "" {
		return 0, 0, "", false
	}
	clean := strings.NewReplacer(" ", "", ",", "", "，", "").Replace(text)
	m := clockCountRe.FindStringSubmatch(clean)
	if m != nil {
		used, err1 := strconv.Atoi(m[1])
		owned, err2 := strconv.Atoi(m[2])
		if err1 == nil && err2 == nil {
			return used, owned, text, true
		}
	}
	logger.Debug(pageTag, "发条数量手动解析失败: %s", text)
	// 原始文本存在但无法解析成分数，直接返回失败，避免兜底误识别。
	return 0, 0, text, false
}

func hasRect(r image.Rectangle) bool { return !r.Empty() }

// TapQuickBattleConfirm 点击快转弹窗确认按钮（对应 BattlePage.tapQuickBattleConfirm）。
func TapQuickBattleConfirm() { touch.TapArea(features.QuickDialog.ConfirmBtn, 1000) }

// TapQuickBattleCancel 点击快转弹窗取消按钮（对应 BattlePage.tapQuickBattleCancel）。
func TapQuickBattleCancel() { touch.TapArea(features.QuickDialog.CancelBtn, 1000) }

// TapSettleUntilBattlePage 点击结算按钮直到战斗页再次出现（对应 BattlePage.tapSettleUntilBattlePage）。
func TapSettleUntilBattlePage() bool {
	if !hasRect(features.SettleBtn) || !hasFeature(features.Feature) {
		logger.Warn(pageTag, " settleBtn / battle feature 未配置")
		return false
	}
	ok, _ := color.TapUntilMatch(features.SettleBtn, features.Feature,
		color.TapOpts{TimeoutMs: 30000, IntervalMs: 500, TapDelayMs: 800, SleepMs: 800})
	return ok
}

// FindBattleCards 查找本页所有战斗卡（对应 BattlePage.findBattleCards）。
func FindBattleCards() []image.Point {
	if !hasFindDef(features.BattleCardFeature) {
		logger.Warn(pageTag, " 战斗卡_特征 未配置")
		return nil
	}
	return color.FindAll(features.BattleCardFeature)
}

// TapBattleCard 点击战斗卡（对应 BattlePage.tapBattleCard）。
func TapBattleCard(pt image.Point) { touch.TapR(pt.X, pt.Y, 1000) }

// RecognizeSoulStoneType 识别灵魂石类型（对应 BattlePage.recognizeSoulStoneType）。
// 同一区域命中多个目标灵魂石时视为无法区分，返回空串。
func RecognizeSoulStoneType(targetNames map[string]bool) string {
	if len(targetNames) == 0 {
		return ""
	}

	type match struct {
		name     string
		category string
		x, y     int
	}
	var matches []match
	for _, category := range soulStoneCategories {
		defs, ok := features.SoulStones[category]
		if !ok {
			continue
		}
		for name, def := range defs {
			if !targetNames[name] || !hasFindDef(def) {
				continue
			}
			if x, y, found := color.Find(def); found {
				matches = append(matches, match{name: name, category: category, x: x, y: y})
			}
		}
	}

	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		logger.Debug(pageTag, "灵魂石匹配 %s/%s", matches[0].category, matches[0].name)
		return matches[0].name
	}
	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		parts = append(parts, fmt.Sprintf("%s/%s(%d,%d)", m.category, m.name, m.x, m.y))
	}
	logger.Warn(pageTag, "灵魂石多个候选命中，无法区分: %s", strings.Join(parts, " , "))
	return ""
}

// SwipeUpAndCheckLastPage 向上滑动并识别是否已到末页（对应 BattlePage.swipeUpAndCheckLastPage）。
func SwipeUpAndCheckLastPage() bool {
	swipe := features.PageSwipe
	if swipe.X1 == 0 && swipe.Y1 == 0 && swipe.X2 == 0 && swipe.Y2 == 0 {
		logger.Warn(pageTag, " 翻页滑动 未配置")
		return true
	}
	if !hasFindDef(features.LastPageFeature) {
		logger.Warn(pageTag, " 末页_特征 未配置")
		return true
	}

	isLastPage := false
	touch.SwipeEx(touch.SwipeOpts{
		X1: swipe.X1, Y1: swipe.Y1, X2: swipe.X2, Y2: swipe.Y2,
		MoveMs: 500, HoldMs: 200, DownMs: 50, UpMs: 500,
		BeforeUp: func() {
			if _, _, ok := color.Find(features.LastPageFeature); ok {
				isLastPage = true
			}
		},
	})

	logger.Info(pageTag, "翻页 hold 识别末页=%v", isLastPage)
	return isLastPage
}
