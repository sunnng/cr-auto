// Package jelly 对应 Lua 工程的 game/常规_未知的地底矿山/模块_解除洋菜冻/：解除洋菜冻页面与会话。
package jelly

import (
	"regexp"
	"strconv"
	"strings"

	"app/internal/game/mine"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
)

const pageTag = "[解除洋菜冻.页面]"

var features = mine.Jelly()

var configFeatures = features.Config

// IsJellyPage 判断是否在解除洋菜冻页面（对应 JellyPage.isJellyPage）。
func IsJellyPage() bool { return color.Match(features.Feature) }

// WaitJellyPage 等待解除洋菜冻页面出现（对应 JellyPage.waitJellyPage，默认 30s）。
func WaitJellyPage(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(features.Feature, timeoutMs, intervalMs, 800)
}

// CanClaimAll 是否可全部领取（对应 JellyPage.canClaimAll）。
func CanClaimAll() bool { return color.Match(features.ClaimAllFeature) }

// TapClaimAll 点击全部领取（对应 JellyPage.tapClaimAll）。
func TapClaimAll() { touch.TapArea(features.ClaimAllBtn, 800) }

// TapSettle 点击结算区域（对应 JellyPage.tapSettle）。
func TapSettle() { touch.TapArea(features.SettleBtn, 800) }

// TapBack 点击返回（对应 JellyPage.tapBack）。
func TapBack() { touch.TapArea(features.BackBtn, 1000) }

// FindConfigBtn OCR 查找「配置」按钮坐标（对应 JellyPage.findConfigBtn）。
func FindConfigBtn() (int, int, bool) {
	return ocr.Find("配置", features.OcrRegion)
}

// TapConfigBtn 点击配置按钮（对应 JellyPage.tapConfigBtn）。
func TapConfigBtn(x, y int) bool {
	if x < 0 || y < 0 {
		return false
	}
	touch.TapR(x, y, 800)
	return true
}

// TapEnterBtn 点击矿山相关入口的「解除洋菜冻」按钮（对应 JellyPage.tapEnterBtn）。
func TapEnterBtn() {
	touch.TapArea(mine.MineVenture().JellyBtn, 1000)
}

// IsConfigPage 判断是否在配置洋菜冻界面（对应 JellyPage.isConfigPage）。
func IsConfigPage() bool { return color.Match(configFeatures.Feature) }

// WaitConfigPage 等待配置洋菜冻界面出现（对应 JellyPage.waitConfigPage，默认 30s）。
func WaitConfigPage(timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if intervalMs <= 0 {
		intervalMs = 500
	}
	return color.WaitMatch(configFeatures.Feature, timeoutMs, intervalMs, 800)
}

// CanChoose 配置界面是否可选择（对应 JellyPage.canChoose）。
func CanChoose() bool { return color.Match(configFeatures.Chooseable) }

// TapChoose 点击选择按钮（对应 JellyPage.tapChoose）。
func TapChoose() { touch.TapArea(configFeatures.ChooseBtn, 800) }

// TapConfigBack 点击配置界面返回（对应 JellyPage.tapConfigBack）。
func TapConfigBack() { touch.TapArea(configFeatures.BackBtn, 1000) }

var (
	remainDaysRe    = regexp.MustCompile(`(\d+)\s*天`)
	remainHoursRe   = regexp.MustCompile(`(\d+)\s*小时`)
	remainMinutesRe = regexp.MustCompile(`(\d+)\s*分钟`)
	remainSecondsRe = regexp.MustCompile(`(\d+)\s*秒`)
)

// ParseRemainTimeText 从文本中解析中文时长（对应 Lua parseRemainTimeText）。
// 支持 X天Y小时Z分钟W秒 等组合；无法解析时返回 ok=false。
func ParseRemainTimeText(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	match := func(re *regexp.Regexp) int {
		m := re.FindStringSubmatch(text)
		if m == nil {
			return 0
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return 0
		}
		return n
	}
	days := match(remainDaysRe)
	hours := match(remainHoursRe)
	minutes := match(remainMinutesRe)
	seconds := match(remainSecondsRe)
	if days == 0 && hours == 0 && minutes == 0 && seconds == 0 {
		return 0, false
	}
	return days*86400 + hours*3600 + minutes*60 + seconds, true
}

// ReadRemainTime OCR 识别解除洋菜冻剩余时间（对应 JellyPage.readRemainTime）。
func ReadRemainTime() (int, bool) {
	r, ok := ocr.Scan(features.OcrRegion, ocr.MultiMode, ocr.ReturnTypeJSON)
	if !ok {
		logger.Warn(pageTag, " readRemainTime: OCR 扫描失败")
		return 0, false
	}

	// 优先从合并 text 解析。
	if remain, ok := ParseRemainTimeText(r.Text); ok {
		logger.Info(pageTag, "readRemainTime: 识别到剩余时间 %ds", remain)
		return remain, true
	}

	// 兜底：逐 item 解析。
	for _, item := range r.Items {
		if remain, ok := ParseRemainTimeText(item.Words); ok {
			logger.Info(pageTag, "readRemainTime: 从 item 识别到剩余时间 %ds", remain)
			return remain, true
		}
	}

	logger.Warn(pageTag, "readRemainTime: 未识别到剩余时间，raw=%s", strings.TrimSpace(r.Raw))
	return 0, false
}
