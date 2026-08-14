// Package dialog 对应 Lua 工程的 lib/dialog.lua：通用弹窗对象，
// 统一识别、确认/取消/不再显示点击，支持 ifVisible（守卫）与 flow（业务流程）两种模式；
// ResolveUntilIdle 轮询消解顺序未知的双弹窗。
package dialog

import (
	"fmt"
	"image"
	"sort"
	"time"

	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/touch"
	"app/internal/vision"
)

const defaultTag = "[Dialog]"

const (
	modeIfVisible = "ifVisible"
	modeFlow      = "flow"
)

// Target 弹窗按钮：image.Rectangle（区域）或 image.Point（坐标）。
type Target any

// Def 标准化弹窗定义（对应 Lua normalizeDef 后的字段）。
type Def struct {
	Name          string
	Feature       vision.Feature
	ConfirmBtn    Target
	CancelBtn     Target
	NeverAgainBtn Target
}

// HandleOpts 处理选项（对应 Lua mergeHandleOpts）。
type HandleOpts struct {
	Mode         string // "flow" | "ifVisible"，缺省 "flow"
	Action       string // "confirm" | "cancel"，缺省 "confirm"
	NeverAgain   bool
	WaitAppearMs int  // >0 时先轮询弹窗出现（flow 模式）
	WaitGoneMs   int  // >0 显式等待消失；<0 表示不等待；0 按模式缺省（flow 3000 / ifVisible 不等待）
	Required     bool // WaitAppearMs 超时是否致命
	TapDelayMs   int  // 缺省 800
	IntervalMs   int  // 缺省 500
}

// Dialog 弹窗对象。
type Dialog struct {
	Def Def
	Tag string
}

// New 创建弹窗对象。
func New(def Def, tag string) *Dialog {
	if tag == "" {
		tag = defaultTag
	}
	return &Dialog{Def: def, Tag: tag}
}

func (d *Dialog) name() string {
	if d.Def.Name != "" {
		return d.Def.Name
	}
	return "未命名弹窗"
}

func (d *Dialog) logf(level logger.Level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	switch level {
	case logger.LevelDebug:
		logger.Debug(d.Tag, "%s", message)
	case logger.LevelInfo:
		logger.Info(d.Tag, "%s", message)
	case logger.LevelWarn:
		logger.Warn(d.Tag, "%s", message)
	}
}

func (d *Dialog) warn(format string, args ...any) { d.logf(logger.LevelWarn, format, args...) }

// IsVisible 当前是否可见（对应 Dialog:isVisible）。
func (d *Dialog) IsVisible() bool {
	return color.Match(d.Def.Feature)
}

// WaitAppear 轮询直到弹窗出现（对应 Dialog:waitAppear）。
func (d *Dialog) WaitAppear(timeoutMs, intervalMs int) bool {
	return color.WaitMatch(d.Def.Feature, timeoutMs, intervalMs, 0)
}

// WaitGone 轮询直到弹窗消失（对应 Dialog:waitGone）。
func (d *Dialog) WaitGone(timeoutMs, intervalMs int) bool {
	return color.WaitGone(d.Def.Feature, timeoutMs, intervalMs)
}

// TapNeverAgain 点击“今日不再询问”（对应 Dialog:tapNeverAgain）。
func (d *Dialog) TapNeverAgain(delayMs int) bool {
	if d.Def.NeverAgainBtn == nil {
		return false
	}
	return performTap(d.Def.NeverAgainBtn, delayMs)
}

// TapConfirm 点击确认按钮（对应 Dialog:tapConfirm）。
func (d *Dialog) TapConfirm(delayMs int) (bool, string) { return d.Tap("confirm", delayMs) }

// TapCancel 点击取消按钮（对应 Dialog:tapCancel）。
func (d *Dialog) TapCancel(delayMs int) (bool, string) { return d.Tap("cancel", delayMs) }

// Tap 点击指定动作按钮（对应 Dialog:tap）。
func (d *Dialog) Tap(action string, delayMs int) (bool, string) {
	if delayMs <= 0 {
		delayMs = 800
	}
	if action != "confirm" && action != "cancel" {
		d.warn("无效 action=%s", action)
		return false, "invalid_action"
	}
	btn := d.Def.ConfirmBtn
	reason := "no_confirm_btn"
	if action == "cancel" {
		btn = d.Def.CancelBtn
		reason = "no_cancel_btn"
	}
	if btn == nil {
		d.warn("%s | %s", reason, d.name())
		return false, reason
	}
	if !performTap(btn, delayMs) {
		return false, reason
	}
	return true, ""
}

// handle 合并处理选项（对应 Lua mergeHandleOpts）。
func mergeHandleOpts(opts HandleOpts, defaultMode string) HandleOpts {
	if opts.Mode == "" {
		opts.Mode = defaultMode
		if opts.Mode == "" {
			opts.Mode = modeFlow
		}
	}
	if opts.Action == "" {
		opts.Action = "confirm"
	}
	if opts.TapDelayMs <= 0 {
		opts.TapDelayMs = 800
	}
	if opts.IntervalMs <= 0 {
		opts.IntervalMs = 500
	}
	return opts
}

// tapAndWaitGone 点击动作并（按选项）等待弹窗消失（对应 Dialog:_tapAndWaitGone）。
func (d *Dialog) tapAndWaitGone(opts HandleOpts) (bool, string) {
	if opts.NeverAgain {
		d.TapNeverAgain(opts.TapDelayMs)
	}
	tapped, reason := d.Tap(opts.Action, opts.TapDelayMs)
	if !tapped {
		return false, reason
	}
	d.logf(logger.LevelInfo, "%s 已处理 [%s] action=%s", d.Tag, d.name(), opts.Action)

	if opts.WaitGoneMs < 0 {
		return true, ""
	}
	goneMs := opts.WaitGoneMs
	if goneMs == 0 {
		if opts.Mode == modeFlow {
			goneMs = 3000
		} else {
			return true, ""
		}
	}
	if d.WaitGone(goneMs, opts.IntervalMs) {
		return true, ""
	}
	return false, "not_gone"
}

// Handle 处理弹窗（对应 Dialog:handle）。
func (d *Dialog) Handle(opts HandleOpts) (bool, string) {
	opts = mergeHandleOpts(opts, "")
	name := d.name()

	if opts.Mode == modeIfVisible {
		if !d.IsVisible() {
			return true, ""
		}
		d.logf(logger.LevelInfo, "%s [ifVisible] 命中 [%s]", d.Tag, name)
		return d.tapAndWaitGone(opts)
	}

	// mode == flow
	if opts.WaitAppearMs > 0 {
		if !d.WaitAppear(opts.WaitAppearMs, opts.IntervalMs) {
			if opts.Required {
				d.warn("[flow] 等待超时 [%s]", name)
				return false, "not_visible"
			}
			d.logf(logger.LevelDebug, "%s [flow] 未出现，跳过 [%s]", d.Tag, name)
			return true, "skipped"
		}
	} else if !d.IsVisible() {
		return true, ""
	}

	d.logf(logger.LevelInfo, "%s [flow] 处理 [%s]", d.Tag, name)
	return d.tapAndWaitGone(opts)
}

// ToGuardHandler 生成守卫处理函数（对应 Dialog:toGuardHandler，mode=ifVisible）。
func (d *Dialog) ToGuardHandler(opts HandleOpts) func() {
	opts = mergeHandleOpts(opts, modeIfVisible)
	return func() { d.Handle(opts) }
}

// ChainItem 顺序确定弹窗链的一项（对应 Dialog.handleChain）。
type ChainItem struct {
	Def  Def
	Opts HandleOpts
	Tag  string
}

// HandleChain 顺序确定的弹窗链（对应 Dialog.handleChain）。
func HandleChain(items []ChainItem, defaultOpts HandleOpts) (bool, string) {
	for i := range items {
		item := &items[i]
		opts := mergeHandleOpts(defaultOpts, "")
		if item.Opts.Mode != "" {
			opts.Mode = item.Opts.Mode
		}
		if item.Opts.Action != "" {
			opts.Action = item.Opts.Action
		}
		if item.Opts.NeverAgain {
			opts.NeverAgain = true
		}
		if item.Opts.WaitAppearMs != 0 {
			opts.WaitAppearMs = item.Opts.WaitAppearMs
		}
		if item.Opts.WaitGoneMs != 0 {
			opts.WaitGoneMs = item.Opts.WaitGoneMs
		}
		if item.Opts.Required {
			opts.Required = true
		}
		if item.Opts.TapDelayMs != 0 {
			opts.TapDelayMs = item.Opts.TapDelayMs
		}
		if item.Opts.IntervalMs != 0 {
			opts.IntervalMs = item.Opts.IntervalMs
		}
		d := New(item.Def, item.Tag)
		if ok, reason := d.Handle(opts); !ok {
			return false, reason
		}
	}
	return true, ""
}

// Candidate resolveUntilIdle 候选弹窗项。
type Candidate struct {
	Name     string
	Dialog   *Dialog
	Priority int
	When     func() bool // 附加条件，nil 表示总是候选
	Opts     HandleOpts
}

// ResolveOpts 轮询消解选项。
type ResolveOpts struct {
	TimeoutMs  int // 缺省 8000
	IntervalMs int // 缺省 300
	SettleMs   int // 缺省 800
	MinWaitMs  int // 缺省 500
	MaxHandled int // >0 时达到上限即停止
	Tag        string
}

// Summary 消解摘要。
type Summary struct {
	Handled    int
	Names      []string
	Reason     string
	LastReason string
}

func nowMs() int64 { return time.Now().UnixMilli() }

// ResolveUntilIdle 轮询消解顺序未知的 0~N 个弹窗（对应 Dialog.resolveUntilIdle）。
func ResolveUntilIdle(candidates []Candidate, opts ResolveOpts) (bool, Summary) {
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 8000
	}
	if opts.IntervalMs <= 0 {
		opts.IntervalMs = 300
	}
	if opts.SettleMs <= 0 {
		opts.SettleMs = 800
	}
	if opts.MinWaitMs <= 0 {
		opts.MinWaitMs = 500
	}
	tag := opts.Tag
	if tag == "" {
		tag = defaultTag
	}

	sorted := append([]Candidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].Name < sorted[j].Name
	})

	start := nowMs()
	lastVisibleAt := start
	handled := 0
	var names []string
	var lastReason string

	for nowMs() < start+int64(opts.TimeoutMs) {
		// 弹窗消解过程保持原子，不再触发守卫扫描，避免嵌套处理。
		color.SleepMs(opts.IntervalMs)

		hit := findFirstVisibleCandidate(sorted)
		if hit != nil {
			lastVisibleAt = nowMs()
			name := hit.Name
			ok, reason := hit.Dialog.Handle(hit.Opts)
			lastReason = reason
			if !ok {
				logger.Warn(defaultTag, "%s resolveUntilIdle 失败 [%s] | %s", tag, name, reason)
				return false, Summary{Handled: handled, Names: names, Reason: reason, LastReason: lastReason}
			}
			handled++
			names = append(names, name)
			logger.Info(defaultTag, "%s resolveUntilIdle 已处理 [%s] %d/%d", tag, name, handled, max(opts.MaxHandled, 0))
			if opts.MaxHandled > 0 && handled >= opts.MaxHandled {
				break
			}
		} else {
			elapsed := nowMs() - start
			idleFor := nowMs() - lastVisibleAt
			if elapsed >= int64(opts.MinWaitMs) && idleFor >= int64(opts.SettleMs) {
				break
			}
		}
	}

	if handled == 0 {
		logger.Debug(defaultTag, "%s resolveUntilIdle 无弹窗", tag)
	} else {
		logger.Info(defaultTag, "%s resolveUntilIdle 完成 handled=%d", tag, handled)
	}
	return true, Summary{Handled: handled, Names: names, LastReason: lastReason}
}

func findFirstVisibleCandidate(sorted []Candidate) *Candidate {
	for i := range sorted {
		c := &sorted[i]
		if (c.When == nil || c.When()) && c.Dialog.IsVisible() {
			return c
		}
	}
	return nil
}

// AfterPrimaryOpts resolveAfterPrimary 选项。
type AfterPrimaryOpts struct {
	Primary       ChainItem    // 主弹窗（flow 模式）
	Watch         []AfterWatch // 轮询的分支弹窗
	SuccessWhen   func() bool  // 成功条件
	TimeoutMs     int          // 缺省 5000
	IntervalMs    int          // 缺省 300
	SuccessResult string       // 缺省 "ok"
	Tag           string
}

// AfterWatch 主弹窗后的分支观察项。
type AfterWatch struct {
	Dialog *Dialog
	Opts   HandleOpts
	Result string // 命中后返回的结果串
	After  func() // 命中后回调
}

// ResolveAfterPrimary 先处理主弹窗 flow，再轮询 watch 分支或 successWhen 成功条件
// （对应 Dialog.resolveAfterPrimary）。
func ResolveAfterPrimary(cfg AfterPrimaryOpts) (bool, string, string) {
	tag := cfg.Tag
	if tag == "" {
		tag = defaultTag
	}
	if cfg.TimeoutMs <= 0 {
		cfg.TimeoutMs = 5000
	}
	if cfg.IntervalMs <= 0 {
		cfg.IntervalMs = 300
	}
	successResult := cfg.SuccessResult
	if successResult == "" {
		successResult = "ok"
	}

	primaryOpts := mergeHandleOpts(cfg.Primary.Opts, modeFlow)
	if cfg.Primary.Opts.WaitAppearMs != 0 {
		primaryOpts.WaitAppearMs = cfg.Primary.Opts.WaitAppearMs
	}
	primaryDialog := New(cfg.Primary.Def, cfg.Primary.Tag)
	ok, reason := primaryDialog.Handle(primaryOpts)
	if !ok {
		logger.Warn(defaultTag, "%s resolveAfterPrimary 主弹窗失败 | %s", tag, reason)
		return false, "failed", reason
	}

	deadline := nowMs() + int64(cfg.TimeoutMs)
	for nowMs() < deadline {
		for _, w := range cfg.Watch {
			if w.Dialog.IsVisible() {
				wok, wreason := w.Dialog.Handle(w.Opts)
				if !wok {
					logger.Warn(defaultTag, "%s resolveAfterPrimary watch 失败 | %s", tag, wreason)
					return false, "failed", wreason
				}
				if w.After != nil {
					w.After()
				}
				result := w.Result
				if result == "" {
					result = "watch"
				}
				return true, result, ""
			}
		}

		if cfg.SuccessWhen != nil && cfg.SuccessWhen() {
			return true, successResult, ""
		}

		anyWatchVisible := false
		for _, w := range cfg.Watch {
			if w.Dialog.IsVisible() {
				anyWatchVisible = true
				break
			}
		}
		if !primaryDialog.IsVisible() && !anyWatchVisible {
			return true, successResult, ""
		}

		color.Sleep(cfg.IntervalMs, cfg.IntervalMs)
	}

	logger.Warn(defaultTag, "%s resolveAfterPrimary 轮询超时", tag)
	return false, "failed", "timeout"
}

// performTap 点击目标：区域或坐标点（对应 Lua performTap）。
func performTap(target Target, tapDelayMs int) bool {
	switch t := target.(type) {
	case image.Point:
		touch.TapR(t.X, t.Y, tapDelayMs)
		return true
	case image.Rectangle:
		if t.Empty() {
			return false
		}
		touch.TapArea(t, tapDelayMs)
		return true
	default:
		return false
	}
}
