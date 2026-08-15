// Package seaside 对应 Lua 工程的 game/常规_海滩交易所/：交易所坐标库、页面、会话、路由与任务。
package seaside

import (
	"image"
	"sort"
	"strconv"
	"strings"

	"app/internal/lib/color"
	"app/internal/lib/dialog"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
	"app/internal/vision"
)

const pageTag = "[海滩交易所.页面]"

const (
	dedupRadius      = 80
	defaultMaxSwipes = 20
)

// hasFeature 特征是否已配置（对应 Lua hasFeature）。
func hasFeature(f vision.Feature) bool { return f.Points != "" }

// StockKeys 全部已配置商品键名，排序后返回（对应 MarketPage.stockKeys）。
func StockKeys() []string {
	keys := make([]string, 0, len(seasideFeatures.Stock))
	for key, def := range seasideFeatures.Stock {
		if hasFeatureDef(def) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func hasFeatureDef(def vision.FindDef) bool {
	return def.FirstColor != "" && !def.Region.Empty()
}

// IsCurrent 是否在交易所页（对应 MarketPage.isCurrent）。
func IsCurrent() bool {
	return hasFeature(seasideFeatures.Page.Feature) && color.Match(seasideFeatures.Page.Feature)
}

// WaitCurrent 等待交易所页出现（对应 MarketPage.waitCurrent，默认 30s）。
func WaitCurrent(timeoutMs, intervalMs int) bool {
	if !hasFeature(seasideFeatures.Page.Feature) {
		logger.Warn(pageTag, " page.feature 未配置")
		return false
	}
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(seasideFeatures.Page.Feature, timeoutMs, intervalMs, 1000)
}

// TapClose 关闭交易所页（对应 MarketPage.tapClose，默认 1000ms）。
func TapClose(delayMs int) {
	touch.TapArea(seasideFeatures.Page.CloseBtn, defaultDelay(delayMs, 1000))
}

// EnsureItemTab 确保道具交易所标签选中（对应 MarketPage.ensureItemTab）。
func EnsureItemTab() bool {
	tab := seasideFeatures.Tab.ItemExchangeTab
	if tab.Area.Empty() {
		return IsCurrent()
	}
	touch.TapArea(tab.Area, 800)
	return IsCurrent()
}

func hasNextPage() bool {
	return hasArrowRight() && func() bool {
		_, ok := color.FindPoint(seasideFeatures.List.ArrowRight)
		return ok
	}()
}

func hasArrowRight() bool {
	def := seasideFeatures.List.ArrowRight
	return def.FirstColor != "" && !def.Region.Empty()
}

// IsLastPage 是否已到最后一页（对应 MarketPage.isLastPage）。
func IsLastPage() bool {
	return hasArrowRight() && !hasNextPage()
}

// SwipeNextPage 左滑翻到下一页（对应 MarketPage.swipeNextPage）。
func SwipeNextPage() bool {
	if IsLastPage() {
		logger.Info(pageTag, " 右箭头不可见，列表已到右侧尽头")
		return false
	}
	swipe := seasideFeatures.List.Swipe
	touch.SwipeEx(touch.SwipeOpts{
		X1: swipe.X1, Y1: swipe.Y1, X2: swipe.X2, Y2: swipe.Y2,
		HoldMs: swipe.HoldMs, UpMs: swipe.UpMs,
	})
	color.Sleep(700, 300)
	return true
}

func slotBtnCenter(pt image.Point) (int, int) {
	return pt.X, pt.Y + seasideFeatures.Slot.BuyBtnOffsetY
}

func slotTapRect(pt image.Point) image.Rectangle {
	cx, cy := slotBtnCenter(pt)
	halfW := seasideFeatures.Slot.BuyBtnHalfW
	halfH := seasideFeatures.Slot.BuyBtnHalfH
	return image.Rect(cx-halfW, cy-halfH, cx+halfW, cy+halfH)
}

func slotCrateRect(pt image.Point) image.Rectangle {
	halfW := seasideFeatures.Slot.CrateHalfW
	halfH := seasideFeatures.Slot.CrateHalfH
	cy := pt.Y + seasideFeatures.Slot.CrateOffsetY
	return image.Rect(pt.X-halfW, cy-halfH, pt.X+halfW, cy+halfH)
}

// IsSlotSoldOut 槽位是否售罄（对应 MarketPage.isSlotSoldOut）。
func IsSlotSoldOut(pt image.Point) bool {
	return ocr.Has("售罄", slotCrateRect(pt))
}

// IsConfirmDialog 是否在购买确认弹窗（对应 MarketPage.isConfirmDialog）。
func IsConfirmDialog() bool {
	return hasFeature(seasideFeatures.Dialog.Feature) && color.Match(seasideFeatures.Dialog.Feature)
}

// WaitConfirmDialog 等待确认弹窗出现（对应 MarketPage.waitConfirmDialog，默认 5s）。
func WaitConfirmDialog(timeoutMs, intervalMs int) bool {
	if !hasFeature(seasideFeatures.Dialog.Feature) {
		logger.Warn(pageTag, " confirmDialog.feature 未配置")
		return false
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if intervalMs <= 0 {
		intervalMs = 300
	}
	return color.WaitMatch(seasideFeatures.Dialog.Feature, timeoutMs, intervalMs, 0)
}

// TapDialogClose 关闭确认弹窗（对应 MarketPage.tapDialogClose）。
func TapDialogClose(delayMs int) bool {
	btn := seasideFeatures.Dialog.CancelBtn
	if btn.Empty() {
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 800))
	return true
}

// IsItemShortageDialog 是否在道具不足弹窗（对应 MarketPage.isItemShortageDialog）。
func IsItemShortageDialog() bool {
	return hasFeature(seasideFeatures.Shortage.Feature) && color.Match(seasideFeatures.Shortage.Feature)
}

// TapItemShortageCancel 关闭道具不足弹窗（对应 MarketPage.tapItemShortageCancel）。
func TapItemShortageCancel(delayMs int) bool {
	btn := seasideFeatures.Shortage.CancelBtn
	if btn.Empty() {
		logger.Warn(pageTag, " itemShortageDialog.cancelBtn 未配置")
		return false
	}
	touch.TapArea(btn, defaultDelay(delayMs, 800))
	return true
}

// TapShelfAndResolve 点击货架并消解确认/道具不足弹窗（对应 MarketPage.tapShelfAndResolve）。
// 返回 "purchased" / "shortage" / "failed"。
func TapShelfAndResolve(pt image.Point) string {
	confirmDialog := dialog.New(dialog.Def{
		Name:       "购买确认",
		Feature:    seasideFeatures.Dialog.Feature,
		ConfirmBtn: seasideFeatures.Dialog.ConfirmBtn,
		CancelBtn:  seasideFeatures.Dialog.CancelBtn,
	}, pageTag)
	shortageDialog := dialog.New(dialog.Def{
		Name:      "道具不足",
		Feature:   seasideFeatures.Shortage.Feature,
		CancelBtn: seasideFeatures.Shortage.CancelBtn,
	}, pageTag)

	touch.TapArea(slotTapRect(pt), 800)

	ok, outcome, reason := dialog.ResolveAfterPrimary(dialog.AfterPrimaryOpts{
		Primary: dialog.ChainItem{
			Def: confirmDialog.Def,
			Opts: dialog.HandleOpts{
				Mode: "flow", Action: "confirm",
				WaitAppearMs: 5000, Required: true, IntervalMs: 300,
			},
			Tag: pageTag,
		},
		Watch: []dialog.AfterWatch{{
			Dialog: shortageDialog,
			Opts: dialog.HandleOpts{
				Action: "cancel", WaitGoneMs: 2000, IntervalMs: 300,
			},
			Result: "shortage",
			After: func() {
				if confirmDialog.IsVisible() {
					confirmDialog.Handle(dialog.HandleOpts{
						Mode: "ifVisible", Action: "cancel", WaitGoneMs: 3000, IntervalMs: 300,
					})
				}
			},
		}},
		SuccessWhen:   func() bool { return !confirmDialog.IsVisible() },
		SuccessResult: "purchased",
		TimeoutMs:     5000,
		IntervalMs:    300,
		Tag:           pageTag,
	})

	if !ok {
		if reason == "not_visible" {
			logger.Warn(pageTag, " 点击货架后确认弹窗未出现")
		} else {
			logger.Warn(pageTag, " 购买确认后结果未知，尝试关闭确认弹窗 | %s", reason)
			confirmDialog.Handle(dialog.HandleOpts{
				Mode: "ifVisible", Action: "cancel", WaitGoneMs: 3000, IntervalMs: 300,
			})
		}
		return "failed"
	}

	if outcome == "shortage" {
		logger.Info(pageTag, " 命中道具不足弹窗，取消本次购买")
	}
	return outcome
}

// IsFreeRefresh 是否可免费刷新（对应 MarketPage.isFreeRefresh）。
func IsFreeRefresh() bool {
	rect := seasideFeatures.Page.RefreshOcr
	if rect.Empty() {
		rect = seasideFeatures.Page.RefreshStatusOcr
	}
	return !rect.Empty() && ocr.Has("免费刷新", rect)
}

// ReadRestockSeconds 读取补货倒计时（对应 MarketPage.readRestockSeconds）。
// 返回 (restockSec, raw)；ok=false 表示未能读取，restockSec==0 表示可免费刷新。
func ReadRestockSeconds() (restockSec int, raw string, ok bool) {
	rect := seasideFeatures.Page.RefreshOcr
	if rect.Empty() {
		rect = seasideFeatures.Page.RefreshStatusOcr
	}
	if rect.Empty() {
		logger.Warn(pageTag, " refreshOcr 未配置")
		return 0, "", false
	}
	text := ocr.Text(rect)
	if text == "" {
		return 0, "", false
	}
	if strings.Contains(text, "免费刷新") {
		return 0, text, true
	}
	// 解析 h:m:s。
	parts := strings.Split(text, ":")
	if len(parts) == 3 {
		h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		s, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err1 == nil && err2 == nil && err3 == nil {
			return h*3600 + m*60 + s, text, true
		}
	}
	logger.Warn(pageTag, " 补货倒计时 OCR 解析失败: %s", text)
	return 0, text, false
}

// TapRefresh 点击刷新（对应 MarketPage.tapRefresh）。
func TapRefresh() bool {
	if seasideFeatures.Page.RefreshBtn.Empty() {
		logger.Warn(pageTag, " refreshBtn 未配置")
		return false
	}
	touch.TapArea(seasideFeatures.Page.RefreshBtn, 1200)
	color.Sleep(1000, 300)
	if !seasideFeatures.Page.RefreshOcr.Empty() || !seasideFeatures.Page.RefreshStatusOcr.Empty() {
		color.WaitGone(func() bool { return IsFreeRefresh() }, 10000, 500)
	}
	return true
}

// configuredItems 过滤已配置的商品定义（对应 Lua configuredItems）。
func configuredItems(itemKeys []string) []stockItem {
	var out []stockItem
	for _, key := range itemKeys {
		def, ok := seasideFeatures.Stock[key]
		if ok && hasFeatureDef(def) {
			out = append(out, stockItem{key: key, def: def})
		} else {
			logger.Warn(pageTag, " 未配置 Stock: %s", key)
		}
	}
	return out
}

type stockItem struct {
	key string
	def vision.FindDef
}

func sortByX(points []image.Point) []image.Point {
	sorted := append([]image.Point(nil), points...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].X < sorted[j].X })
	return sorted
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

func collectVisibleTargets(itemDefs []stockItem) []image.Point {
	var points []image.Point
	var seen []image.Point
	for _, item := range itemDefs {
		found := color.FindAll(item.def)
		for _, pt := range found {
			if !isNearExisting(pt, seen, dedupRadius) {
				seen = append(seen, pt)
				points = append(points, pt)
			}
		}
	}
	return sortByX(points)
}

// PurchaseStats 扫货统计（对应 Lua purchaseWishlist 返回 stats）。
type PurchaseStats struct {
	Purchased int
	Skipped   struct {
		SoldOut  int
		Shortage int
		Failed   int
	}
}

// PurchaseWishlist 扫描并购买愿望清单商品（对应 MarketPage.purchaseWishlist）。
func PurchaseWishlist(itemKeys []string) PurchaseStats {
	var stats PurchaseStats
	itemDefs := configuredItems(itemKeys)
	if len(itemDefs) == 0 {
		logger.Warn(pageTag, " 无可购买道具配置")
		return stats
	}

	maxSwipes := seasideFeatures.List.MaxSwipes
	if maxSwipes <= 0 {
		maxSwipes = defaultMaxSwipes
	}

	swipes := 0
	for swipes <= maxSwipes {
		var visited []image.Point
		points := collectVisibleTargets(itemDefs)
		logger.Info(pageTag, " 扫描可见商品 目标命中:%d 滑动:%d", len(points), swipes)
		for _, pt := range points {
			if isNearExisting(pt, visited, dedupRadius) {
				continue
			}
			visited = append(visited, pt)
			if IsSlotSoldOut(pt) {
				stats.Skipped.SoldOut++
				logger.Info(pageTag, " 商品已售罄，跳过")
				continue
			}
			logger.Info(pageTag, " 尝试购买")
			result := TapShelfAndResolve(pt)
			switch result {
			case "purchased":
				stats.Purchased++
			case "shortage":
				stats.Skipped.Shortage++
			default:
				stats.Skipped.Failed++
			}
		}
		if IsLastPage() {
			logger.Info(pageTag, " 已是最后一页，结束扫货")
			break
		}
		if !SwipeNextPage() {
			break
		}
		swipes++
	}
	return stats
}

func defaultDelay(delayMs, fallback int) int {
	if delayMs <= 0 {
		return fallback
	}
	return delayMs
}
