// Package biscuit 对应 Lua 工程的 game/功能_洗脆饼/：脆饼词条参考表与洗脆饼任务。
package biscuit

import (
	"image"
	"sort"
	"strconv"
	"strings"

	"app/internal/config"
	"app/internal/core"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/status"
	"app/internal/lib/touch"
	"app/internal/lib/userconfig"
	"app/internal/vision"
)

const taskTag = "[洗脆饼词条]"

// Effect 一条识别出的词条（对应 Lua readEffects 返回表项）。
type Effect struct {
	Name  string
	Value float64
	Raw   string
}

// extractNumber 从字符串末尾反向提取数字（支持小数），返回 (value, name)（对应 Lua extractNumber）。
func extractNumber(str string) (float64, string, bool) {
	if str == "" {
		return 0, "", false
	}
	runes := []rune(str)
	end := len(runes)
	start := end
	for i := end - 1; i >= 0; i-- {
		c := runes[i]
		if (c >= '0' && c <= '9') || c == '.' {
			start = i
		} else {
			break
		}
	}
	if start >= end {
		return 0, str, false
	}
	numStr := string(runes[start:end])
	name := strings.TrimSpace(string(runes[:start]))
	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, name, false
	}
	return value, name, true
}

// parseRaw 按 % 拆分并解析（对应 Lua parseRaw）。
func parseRaw(text string) []Effect {
	if text == "" {
		return nil
	}
	var result []Effect
	for _, part := range strings.Split(text, "%") {
		if part == "" {
			continue
		}
		value, name, ok := extractNumber(part)
		if ok && name != "" {
			result = append(result, Effect{Name: name, Value: value, Raw: part + "%"})
		}
	}
	return result
}

var effectOcrRect = image.Rect(427, 390, 1162, 760)

// readEffects 读取脆饼副词条，固定 4 条（对应 Lua readEffects）。
func readEffects() []Effect {
	scan, ok := ocr.Scan(effectOcrRect, ocr.MultiMode, ocr.ReturnTypeText)
	raw := ""
	if ok {
		raw = scan.Raw
	}
	result := parseRaw(raw)

	// 脆饼固定 4 条副词条；OCR 多识别时截断，不足时补空。
	if len(result) > 4 {
		result = result[:4]
	}
	for len(result) < 4 {
		result = append(result, Effect{Name: "未知", Value: 0, Raw: ""})
	}
	return result
}

// checkSums 总和规则检查（对应 Lua checkSums）。
func checkSums(effects []Effect, sumRules []config.BiscuitSumRule) (bool, string) {
	if len(sumRules) == 0 {
		return false, "未配置总和规则"
	}

	for _, r := range sumRules {
		if !r.Enabled || r.Name == "" || r.Count <= 0 || r.MinSum <= 0 {
			continue
		}
		need := min(4, r.Count)
		var values []float64
		for _, e := range effects {
			if e.Name == r.Name {
				values = append(values, e.Value)
			}
		}
		if len(values) >= need {
			sort.Sort(sort.Reverse(sort.Float64Slice(values)))
			sum := 0.0
			for i := 0; i < need; i++ {
				sum += values[i]
			}
			if sum >= float64(r.MinSum) {
				return true, formatSumRule(r, need, sum)
			}
		}
	}
	return false, "总和规则未满足"
}

func formatSumRule(r config.BiscuitSumRule, need int, sum float64) string {
	return "[" + r.Name + "]取高" + strconv.Itoa(need) + "条 总和" +
		strconv.FormatFloat(sum, 'f', 1, 64) + "≥" + strconv.Itoa(r.MinSum)
}

// checkSlots 槽位规则检查（对应 Lua checkSlots）。
func checkSlots(effects []Effect, targets []config.BiscuitTarget) (bool, string) {
	// 1. 收集启用的规则（minPercent<=0 视为未配置，对应 Lua 缺失 minPercent 时报错而非放行）。
	var active []struct {
		Name string
		Min  int
	}
	for _, r := range targets {
		if r.Enabled && r.Name != "" && r.MinPercent > 0 {
			active = append(active, struct {
				Name string
				Min  int
			}{Name: r.Name, Min: r.MinPercent})
		}
	}
	if len(active) == 0 {
		return false, "无槽位规则"
	}

	// 2. 复制实际词条，准备标记使用状态。
	type poolItem struct {
		Name  string
		Value float64
		Used  bool
	}
	pool := make([]poolItem, len(effects))
	for i, e := range effects {
		pool[i] = poolItem{Name: e.Name, Value: e.Value}
	}

	// 3. 规则按阈值降序排序（最难满足的优先拿词条）。
	sort.SliceStable(active, func(i, j int) bool { return active[i].Min > active[j].Min })

	// 4. 逐条匹配：每个规则在未使用的实际词条中找第一个满足的。
	for _, rule := range active {
		found := false
		for i := range pool {
			if !pool[i].Used && pool[i].Name == rule.Name && pool[i].Value >= float64(rule.Min) {
				pool[i].Used = true
				found = true
				break
			}
		}
		if !found {
			return false, "缺[" + rule.Name + ">=" + strconv.Itoa(rule.Min) + "]"
		}
	}
	return true, "毕业"
}

// check 槽位或总和满足其一（对应 Lua check）。
func check(effects []Effect, targets []config.BiscuitTarget, sumRules []config.BiscuitSumRule) (bool, string) {
	ok1, msg1 := checkSlots(effects, targets)
	if ok1 {
		return true, msg1
	}
	ok2, msg2 := checkSums(effects, sumRules)
	if ok2 {
		return true, msg2
	}
	return false, msg1
}

// 洗脆饼重掷按钮与确认弹窗特征（对应 Lua task.lua 内联区域）。
var (
	rerollBtn = image.Rect(914, 815, 961, 851)

	resetDialogFeature = vision.Feature{Points: "1026|627|7ace0e-101010,745|629|0ca6df-101010,863|257|363d5f-101010,782|466|505050-101010,785|419|505050-101010", Sim: 0.9}
	resetNeverAgainBtn = image.Rect(874, 727, 887, 740)
	resetConfirmBtn    = image.Rect(932, 624, 977, 643)

	sameDialogFeature = vision.Feature{Points: "1041|635|7ace0e-101010,711|632|0ca6df-101010,815|263|f70b05-101010,972|257|363d5f-101010,802|248|ffffff-101010,836|440|505050-101010", Sim: 0.9}
	sameNeverAgainBtn = image.Rect(876, 725, 885, 739)
	sameConfirmBtn    = image.Rect(942, 626, 971, 641)
)

// Run 运行洗脆饼词条任务（对应 Task.run）。
func Run(_ *core.Guard) error {
	logger.Info(taskTag, "开始")

	isConfirmResetDialog := false
	isConfirmSameDialog := false

	cfg := biscuitCfg()
	maxRolls := cfg.MaxRolls
	if maxRolls <= 0 {
		logger.Warn(taskTag, " maxRolls 未配置，跳过")
		return core.ErrSkip
	}
	targets := cfg.Targets
	sumRules := cfg.SumRules
	graduated := false

	status.SetBiscuitReroll(status.BiscuitReroll{Current: 0, Max: maxRolls, HasCurrent: true})

	currentRolls := 0
	for currentRolls < maxRolls {
		currentRolls++
		status.SetBiscuitReroll(status.BiscuitReroll{Current: currentRolls, Max: maxRolls, HasCurrent: true})

		effects := readEffects()

		res, msg := check(effects, targets, sumRules)
		if res {
			graduated = true
			status.SetBiscuitReroll(status.BiscuitReroll{Current: currentRolls, Max: maxRolls, HasCurrent: true, Extra: "已毕业"})
			logger.Info(taskTag, "%s", msg)
			// 与 userconfig 段内字段名保持一致（对应 Lua UserConfig.set("biscuit", {enabled=false})）。
			_ = userconfig.Default().Set("biscuit", map[string]any{"Enabled": false})
			break
		}

		touch.TapArea(rerollBtn, 1000)

		// 确认重置弹窗。
		if !isConfirmResetDialog && color.Match(resetDialogFeature) {
			touch.TapArea(resetNeverAgainBtn, 1000)
			isConfirmResetDialog = true
			touch.TapArea(resetConfirmBtn, 1000)
		}

		// 确认相同脆饼弹窗。
		if !isConfirmSameDialog && color.Match(sameDialogFeature) {
			touch.TapArea(sameNeverAgainBtn, 1000)
			isConfirmSameDialog = true
			touch.TapArea(sameConfirmBtn, 1000)
		}
	}

	if !graduated && currentRolls >= maxRolls {
		status.SetBiscuitReroll(status.BiscuitReroll{Current: currentRolls, Max: maxRolls, HasCurrent: true, Extra: "已达上限"})
	}
	return nil
}

func biscuitCfg() config.BiscuitConfig {
	cfg, err := userconfig.Biscuit()
	if err != nil {
		logger.Warn(taskTag, "读取配置失败: %v", err)
		return config.Static.User.Biscuit
	}
	return cfg
}
