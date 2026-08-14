// Package mining 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山开采/：开采页面与会话。
package mining

import (
	"image"
	"sort"
	"strings"
	"time"

	"app/internal/game/mine"
	"app/internal/lib/color"
	"app/internal/lib/dialog"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
	"app/internal/vision"
)

const (
	pageTag            = "[矿山开采.页面]"
	dedupRadius        = 80
	selectedNearRadius = 120
	maxSwipes          = 20
	tapYOffset         = -200
	tapSettleMs        = 450
)

var features = mine.Mining()

// swipeCardListOpts 卡列表水平滑动参数（对应 Lua SWIPE_CARD_LIST）。
var swipeCardListOpts = touch.SwipeOpts{
	X1: 1480, Y1: 738, X2: 150, Y2: 738,
	MoveMs: 600, HoldMs: 1000, DownMs: 50,
}

var lastNoMineCard = false

// hasFeature 特征是否已配置（对应 Lua hasFeature）。
func hasFeature(f vision.Feature) bool { return f.Points != "" }

// hasFindDef 找色定义是否已配置（零值区域/首色视为未配置）。
func hasFindDef(def vision.FindDef) bool { return def.FirstColor != "" }

func hasRect(r image.Rectangle) bool { return !r.Empty() }

func reverseSwipe(opts touch.SwipeOpts) touch.SwipeOpts {
	opts.X1, opts.Y1, opts.X2, opts.Y2 = opts.X2, opts.Y2, opts.X1, opts.Y1
	return opts
}

func sortByX(points []image.Point) {
	sort.Slice(points, func(i, j int) bool { return points[i].X < points[j].X })
}

func isNearExisting(pt image.Point, list []image.Point, radius int) bool {
	r2 := radius * radius
	for _, p := range list {
		dx := pt.X - p.X
		dy := pt.Y - p.Y
		if dx*dx+dy*dy <= r2 {
			return true
		}
	}
	return false
}

// findSelectedCardPoints 找已选矿卡标记点（SelectedMark 未配置时返回空）。
func findSelectedCardPoints() []image.Point {
	if !hasFindDef(features.CardSelect.SelectedMark) {
		return nil
	}
	return color.FindAll(features.CardSelect.SelectedMark)
}

func tapCardPoint(pt image.Point) {
	touch.TapR(pt.X, pt.Y+tapYOffset, tapSettleMs)
}

// tapCardIfQuotaIncreases 点击矿卡并校验配额增加；误触已选卡时恢复（对应 Lua tapCardIfQuotaIncreases）。
func tapCardIfQuotaIncreases(pt image.Point, targetCur int) bool {
	before, _, _, ok := ReadChooseQuota()
	if !ok || before >= targetCur {
		return false
	}
	tapCardPoint(pt)
	after, _, _, ok := ReadChooseQuota()
	if !ok {
		return false
	}
	if after > before {
		logger.Debug(pageTag, "选中 +1 (%d→%d)", before, after)
		return true
	}
	if after < before {
		logger.Warn(pageTag, "误触已选卡 (%d→%d)，恢复", before, after)
		tapCardPoint(pt)
	}
	return false
}

var noMineCardHints = []string{"没有可选择的矿脉卡", "没有"}

func hasNoMineCardHint(text string) bool {
	if text == "" {
		return false
	}
	for _, hint := range noMineCardHints {
		if strings.Contains(text, hint) {
			return true
		}
	}
	return false
}

// ocrRectHasText 区域内是否有非空文字（对应 Lua ocrRectHasText）。
func ocrRectHasText(rect image.Rectangle) bool {
	r, ok := ocr.Scan(rect, ocr.MultiMode, ocr.ReturnTypeJSON)
	if !ok {
		return false
	}
	if hasNonSpace(r.Text) {
		return true
	}
	for _, item := range r.Items {
		if hasNonSpace(item.Words) {
			return true
		}
	}
	return false
}

func hasNonSpace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}
	return false
}

// HasNoMineCardInList 矿脉卡清单是否提示没有矿卡（对应 MiningPage.hasNoMineCardInList）。
func HasNoMineCardInList() bool {
	if !hasRect(features.NoMineCardOcr) {
		return false
	}
	r, ok := ocr.Scan(features.NoMineCardOcr, ocr.MultiMode, ocr.ReturnTypeText)
	if !ok {
		return false
	}
	if hasNoMineCardHint(r.Text) {
		logger.Info(pageTag, "noMineCardOcr 检测到无矿卡提示: %s", r.Text)
		return true
	}
	return false
}

// swipeCardList 滑动卡列表并检查是否已到尽头（对应 Lua swipeCardList）。
func swipeCardList(direction string) bool {
	if direction == "right" {
		touch.SwipeEx(reverseSwipe(swipeCardListOpts))
	} else {
		touch.SwipeEx(swipeCardListOpts)
	}
	color.Sleep(300, 100)

	var edgeRect image.Rectangle
	edgeName := "右缘"
	if direction == "right" {
		edgeRect = features.CardListStartOcr
		edgeName = "左缘"
	} else {
		edgeRect = features.CardListEndOcr
	}
	if !ocrRectHasText(edgeRect) {
		logger.Info(pageTag, "卡列表%s无文字，已到尽头", edgeName)
		return false
	}
	color.Sleep(500, 100)
	return true
}

// IsMiningPage 判断是否在开采页（对应 MiningPage.isMiningPage）。
func IsMiningPage() bool {
	return hasFeature(features.Page.Feature) && color.Match(features.Page.Feature)
}

// WaitMiningPage 等待开采页出现（对应 MiningPage.waitMiningPage，默认 60s）。
func WaitMiningPage(timeoutMs, intervalMs int) bool {
	if !hasFeature(features.Page.Feature) {
		return false
	}
	if timeoutMs <= 0 {
		timeoutMs = 60000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	ok, _ := color.Wait(features.Page.Feature, timeoutMs, intervalMs)
	return ok
}

// IsSetup 判断是否在开采准备页（对应 MiningPage.isSetup）。
func IsSetup() bool {
	return hasFeature(features.SetupFeature) && color.Match(features.SetupFeature)
}

// IsSetupReady 判断准备是否就绪（对应 MiningPage.isSetupReady）。
func IsSetupReady() bool {
	return hasFeature(features.SetupReadyFeature) && color.Match(features.SetupReadyFeature)
}

// WaitSetupReady 等待准备就绪（对应 MiningPage.waitSetupReady，默认 30s）。
func WaitSetupReady(timeoutMs, intervalMs int) bool {
	if !hasFeature(features.SetupReadyFeature) {
		logger.Warn(pageTag, " setupReadyFeature 未配置")
		return false
	}
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	ok, _ := color.Wait(features.SetupReadyFeature, timeoutMs, intervalMs)
	return ok
}

// IsRewardPage 判断是否在奖励页（对应 MiningPage.isRewardPage；
// Lua 的 rewardPage 只有 titleText/titleOcr，无比色特征，故仅用 OCR 判断）。
func IsRewardPage() bool {
	if features.RewardPage.TitleText != "" && hasRect(features.RewardPage.TitleOcr) {
		return ocr.Has(features.RewardPage.TitleText, features.RewardPage.TitleOcr)
	}
	return false
}

// IsSettlementRoute 判断是否在结算路径上（对应 MiningPage.isSettlementRoute）。
func IsSettlementRoute() bool {
	return !mine.IsCurrent() && !IsMiningPage()
}

// TapUntilMatchMiningPage 点击奖励确认直到开采页出现（对应 MiningPage.tapUntilMatchMiningPage）。
func TapUntilMatchMiningPage() bool {
	if !hasRect(features.RewardPage.ConfirmBtn) || !hasFeature(features.Page.Feature) {
		logger.Warn(pageTag, " rewardPage.confirmBtn / page.feature 未配置")
		return false
	}
	ok, _ := color.TapUntilMatch(features.RewardPage.ConfirmBtn, features.Page.Feature,
		color.TapOpts{TimeoutMs: 30000, IntervalMs: 500})
	return ok
}

// HasCompletedTask 是否有已完成的开采任务（对应 MiningPage.hasCompletedTask）。
func HasCompletedTask() bool {
	if !hasFindDef(features.CompletedTask) {
		return false
	}
	_, _, ok := color.Find(features.CompletedTask)
	return ok
}

// TapCompletedSlot 点击已完成槽位（对应 MiningPage.tapCompletedSlot）。
func TapCompletedSlot() bool {
	if !hasFindDef(features.CompletedTask) {
		logger.Warn(pageTag, " completedTask 特征未配置")
		return false
	}
	x, y, ok := color.Find(features.CompletedTask)
	if !ok {
		return false
	}
	touch.TapR(x, y, 500)
	return true
}

// HasFreeSlot 是否有空闲槽位（对应 MiningPage.hasFreeSlot）。
func HasFreeSlot() bool {
	if hasFindDef(features.FreeLocationFeature) {
		if _, _, ok := color.Find(features.FreeLocationFeature); ok {
			return true
		}
	}
	if hasFindDef(features.FreePlusFeature) {
		if _, _, ok := color.Find(features.FreePlusFeature); ok {
			return true
		}
	}
	return false
}

// EnterMultiSelect 进入多选模式（对应 MiningPage.enterMultiSelect）。
func EnterMultiSelect() bool {
	lastNoMineCard = false
	if !hasRect(features.MultiSelectBtn) {
		logger.Warn(pageTag, " multiSelectBtn 未配置")
		return false
	}

	ocrRect := features.MultiSelectBtn
	if hasRect(features.MultiSelectOcr) {
		ocrRect = features.MultiSelectOcr
	}
	deadline := time.Now().UnixMilli() + 30000
	interval := 500
	for time.Now().UnixMilli() < deadline {
		if HasNoMineCardInList() {
			logger.Info(pageTag, " 矿脉卡清单提示无矿卡，退出选卡页面")
			backBtn := features.BackBtn
			if hasRect(features.CardSelect.BackBtn) {
				backBtn = features.CardSelect.BackBtn
			}
			touch.TapArea(backBtn, 1000)
			if WaitMiningPage(30000, 500) {
				lastNoMineCard = true
			} else {
				logger.Warn(pageTag, " 退出选卡页面后未回到矿山开采首页")
			}
			return false
		}

		if ocr.Has("选择多个", ocrRect) {
			touch.TapArea(features.MultiSelectBtn, 1000)
			return true
		}
		if ocr.Has("选择一个", ocrRect) {
			return true
		}
		color.Sleep(interval, interval)
	}

	logger.Warn(pageTag, " enterMultiSelect 等待选卡页面超时")
	return false
}

// WasNoMineCard 上次进入多选是否发现无矿卡（对应 MiningPage.wasNoMineCard）。
func WasNoMineCard() bool { return lastNoMineCard }

// TapFreeSlot 点击空闲槽位并进入多选（对应 MiningPage.tapFreeSlot）。
func TapFreeSlot() bool {
	if !hasFindDef(features.FreeLocationFeature) {
		logger.Warn(pageTag, " freeLocationFeature 未配置")
		return false
	}
	x, y, ok := color.Find(features.FreeLocationFeature)
	if !ok && hasFindDef(features.FreePlusFeature) {
		x, y, ok = color.Find(features.FreePlusFeature)
	}
	if !ok {
		return false
	}
	touch.TapR(x, y, 500)
	return EnterMultiSelect()
}

// ReadChooseQuota 读取可选配额 x/x（对应 MiningPage.readChooseQuota）。
func ReadChooseQuota() (cur, max int, raw string, ok bool) {
	return ocr.Fraction(features.CanChooseNum)
}

// SelectTargetCards 按目标矿卡与数量选卡（对应 MiningPage.selectTargetCards）。
// 返回 (已选数量, 是否扫完无新增)。
func SelectTargetCards(targetDef vision.FindDef, needCount int, direction string) (int, bool) {
	if needCount <= 0 {
		return 0, false
	}
	if !hasFindDef(targetDef) {
		logger.Warn(pageTag, " 目标矿卡特征未配置")
		return 0, true
	}

	startCur, startMax, _, _ := ReadChooseQuota()
	if startCur <= 0 {
		startCur = 0
	}
	if startMax <= 0 {
		startMax = needCount
	}
	targetCur := startCur + needCount
	if direction == "" {
		direction = "left"
	}

	swipes := 0
	exhausted := false
	for swipes <= maxSwipes {
		cur, max, _, ok := ReadChooseQuota()
		if !ok {
			cur, max = startCur, startMax
		}
		if cur >= max || cur >= targetCur {
			return cur - startCur, false
		}

		selectedMarks := findSelectedCardPoints()
		var tappedThisPass []image.Point
		progressed := false
		points := color.FindAll(targetDef)
		sortByX(points)
		logger.Info(pageTag, "扫描目标卡 方向:%s 可见:%d 已选:%d 还需:%d 滑动:%d",
			direction, len(points), cur-startCur, targetCur-cur, swipes)

		for _, pt := range points {
			cur, _, _, ok := ReadChooseQuota()
			if !ok || cur >= targetCur {
				break
			}
			if isNearExisting(pt, tappedThisPass, dedupRadius) {
				continue
			}
			if isNearExisting(pt, selectedMarks, selectedNearRadius) {
				logger.Debug(pageTag, " 跳过已选标记卡")
				continue
			}
			tappedThisPass = append(tappedThisPass, pt)
			if tapCardIfQuotaIncreases(pt, targetCur) {
				progressed = true
			}
		}

		cur, _, _, _ = ReadChooseQuota()
		if cur >= targetCur {
			return cur - startCur, false
		}

		if !progressed {
			if !swipeCardList(direction) {
				exhausted = true
				break
			}
			swipes++
		}
	}

	if swipes > maxSwipes {
		exhausted = true
	}
	finalCur, _, _, ok := ReadChooseQuota()
	if !ok {
		finalCur = startCur
	}
	logger.Warn(pageTag, "选卡不足 %d/%d（滑动%d次）", finalCur-startCur, needCount, swipes)
	return finalCur - startCur, exhausted
}

// ConfirmCardSelection 确认选卡（对应 MiningPage.confirmCardSelection）。
func ConfirmCardSelection() bool {
	if !hasRect(features.CardSelect.ConfirmBtn) {
		logger.Warn(pageTag, " cardSelect.confirmBtn 未配置")
		return false
	}
	touch.TapArea(features.CardSelect.ConfirmBtn, 800)
	return true
}

// HasStartableCard 是否有可开始开采的矿卡（对应 MiningPage.hasStartableCard）。
func HasStartableCard() bool {
	if !hasFindDef(features.StartableCard) {
		return false
	}
	_, _, ok := color.Find(features.StartableCard)
	return ok
}

// TapReadySlot 点击可开始槽位并等待准备页（对应 MiningPage.tapReadySlot）。
func TapReadySlot() bool {
	if !hasFindDef(features.StartableCard) {
		logger.Warn(pageTag, " startableCard 特征未配置")
		return false
	}
	x, y, ok := color.Find(features.StartableCard)
	if !ok {
		return false
	}
	touch.TapR(x-100, y+100, 500)
	if !hasFeature(features.SetupFeature) {
		logger.Warn(pageTag, " setupFeature 未配置")
		return false
	}
	ok, _ = color.Wait(features.SetupFeature, 30000, 500)
	return ok
}

// AutoSelectCookieAndStart 自动选择饼干并开始开采（对应 MiningPage.autoSelectCookieAndStart）。
func AutoSelectCookieAndStart() bool {
	autoBtn := features.AutoSelectCookieBtn
	startBtn := features.ConfirmStartBtn
	if !hasRect(autoBtn) || !hasRect(startBtn) {
		logger.Warn(pageTag, " autoSelectCookieBtn / confirmStartBtn 未配置")
		return false
	}

	touch.TapArea(autoBtn, 500)
	if !WaitSetupReady(0, 0) {
		return false
	}
	touch.TapArea(startBtn, 500)

	cookieDialog := dialog.New(dialog.Def{
		Name:          "confirmCookie",
		Feature:       features.DialogConfirmCookie.Feature,
		ConfirmBtn:    features.DialogConfirmCookie.ConfirmBtn,
		NeverAgainBtn: features.DialogConfirmCookie.TodayNotAskAgain,
	}, pageTag)
	countWarningDialog := dialog.New(dialog.Def{
		Name:          "cookieCountWarning",
		Feature:       features.DialogCookieCountWarning.Feature,
		ConfirmBtn:    features.DialogCookieCountWarning.ConfirmBtn,
		NeverAgainBtn: features.DialogCookieCountWarning.TodayNotAskAgain,
	}, pageTag)

	// 两个弹窗出现顺序未知，使用 resolveUntilIdle 处理。
	ok, summary := dialog.ResolveUntilIdle([]dialog.Candidate{
		{Name: "confirmCookie", Dialog: cookieDialog, Priority: 10,
			Opts: dialog.HandleOpts{Mode: "ifVisible", Action: "confirm", NeverAgain: true, WaitGoneMs: 2000, IntervalMs: 300}},
		{Name: "cookieCountWarning", Dialog: countWarningDialog, Priority: 10,
			Opts: dialog.HandleOpts{Mode: "ifVisible", Action: "confirm", NeverAgain: true, WaitGoneMs: 2000, IntervalMs: 300}},
	}, dialog.ResolveOpts{TimeoutMs: 8000, MinWaitMs: 500, SettleMs: 800, MaxHandled: 2, Tag: pageTag})

	if !ok {
		logger.Warn(pageTag, "饼干弹窗处理失败 | %s", summary.Reason)
		return false
	}
	return true
}

// TapBackBtn 点击返回按钮（对应 MiningPage.tapBackBtn）。
func TapBackBtn() { touch.TapArea(features.BackBtn, 1000) }
