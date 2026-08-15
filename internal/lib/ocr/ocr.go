// Package ocr 对应 Lua 工程的 lib/ocr.lua：OCR 门面，识别引擎由宿主注入。
// 引擎为设备端实现（M2 设备验收阶段接入 TomatoOCR / AutoGo 等价接口），
// 未注入时所有识别返回失败并只告警一次（对应 Lua "引擎未初始化" 行为）。
// 桌面测试注入假引擎即可验证解析逻辑。
package ocr

import (
	"encoding/json"
	"image"
	"regexp"
	"strconv"
	"strings"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/touch"
)

const tag = "[OCR]"

// 识别模式与返回类型（对应 config.lua OCR.ENGINE）。
const (
	LineMode       = 2 // 单行
	MultiMode      = 3 // 多行/找点
	ReturnTypeJSON = "json"
	ReturnTypeText = "text"
	ReturnTypeNum  = "num"
)

// Point 识别结果中的坐标点。
type Point struct{ X, Y int }

// Item 一条识别词条：文字与位置（四角或扁平框）。
type Item struct {
	Words    string
	Location []Point
}

// Result 一次识别结果（对应 Lua Ocr.scan 返回值）。
type Result struct {
	Raw    string
	Text   string
	Items  []Item
	X1, Y1 int // 识别区域左上角（坐标换算基准）
}

// Engine 设备端 OCR 引擎：在 rect 区域识别并返回原始结果串
// （"text"/"num" 为纯文本，"json" 为含 words/location 的 JSON）。
type Engine interface {
	Scan(rect image.Rectangle, mode int, returnType string) (string, error)
	// FindTapPoint 引擎侧找字坐标；未实现时返回 false，由 items 兜底。
	FindTapPoint(text string, rect image.Rectangle) (int, int, bool)
}

var (
	engine         Engine
	warnedNotReady bool
	sleepFn        func(ms int)
)

// SetEngine 注入识别引擎；nil 时识别一律失败。
func SetEngine(e Engine) {
	engine = e
	warnedNotReady = false
}

// SetSleep 注入休眠实现（测试用；未注入时以真实时间等待）。
func SetSleep(fn func(ms int)) { sleepFn = fn }

func sleep(ms int) {
	if sleepFn != nil {
		sleepFn(ms)
		return
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func warnNotReady() {
	if !warnedNotReady {
		logger.Warn(tag, "引擎未初始化，识别返回失败（此告警仅一次）。请检查设备端注入。")
		warnedNotReady = true
	}
}

func validRect(rect image.Rectangle) bool {
	return !rect.Empty()
}

// decode 解析引擎 JSON 结果（words/location）。
func decode(str string) ([]Item, string) {
	var data any
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		return nil, ""
	}
	switch d := data.(type) {
	case map[string]any:
		if w, ok := d["words"].(string); ok {
			return []Item{{Words: w, Location: decodeLocation(d["location"])}}, w
		}
		return nil, ""
	case []any:
		var items []Item
		var parts []string
		for _, v := range d {
			if m, ok := v.(map[string]any); ok {
				w, _ := m["words"].(string)
				items = append(items, Item{Words: w, Location: decodeLocation(m["location"])})
				parts = append(parts, w)
			}
		}
		return items, strings.Join(parts, "")
	default:
		return nil, ""
	}
}

// decodeLocation 解析 location：四角点数组或扁平框。
func decodeLocation(loc any) []Point {
	switch l := loc.(type) {
	case []any:
		if len(l) >= 4 && isNum(l[0]) {
			// 扁平框 [x1,y1,x2,y2]。
			return []Point{{X: intNum(l[0]), Y: intNum(l[1])}, {X: intNum(l[2]), Y: intNum(l[3])}}
		}
		var pts []Point
		for _, p := range l {
			if arr, ok := p.([]any); ok && len(arr) >= 2 {
				pts = append(pts, Point{X: intNum(arr[0]), Y: intNum(arr[1])})
			}
		}
		return pts
	default:
		return nil
	}
}

func isNum(v any) bool { _, ok := v.(float64); return ok }

func intNum(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

// localCenter 计算位置中心（对应 Lua localCenter）。
func localCenter(loc []Point) (int, int, bool) {
	if len(loc) == 0 {
		return 0, 0, false
	}
	minX, maxX, minY, maxY := loc[0].X, loc[0].X, loc[0].Y, loc[0].Y
	for _, p := range loc[1:] {
		minX, maxX = min(minX, p.X), max(maxX, p.X)
		minY, maxY = min(minY, p.Y), max(maxY, p.Y)
	}
	return (minX + maxX) / 2, (minY + maxY) / 2, true
}

// findInItems 在词条中找文字坐标（items 兜底）。
func findInItems(items []Item, text string, x1, y1 int) (int, int, bool) {
	for _, item := range items {
		if item.Words != "" && strings.Contains(item.Words, text) {
			if lx, ly, ok := localCenter(item.Location); ok && lx >= 0 && ly >= 0 {
				return lx + x1, ly + y1, true
			}
		}
	}
	return 0, 0, false
}

// Scan 截图并识别（对应 Lua Ocr.scan）；失败返回 false。
func Scan(rect image.Rectangle, mode int, returnType string) (*Result, bool) {
	if engine == nil {
		warnNotReady()
		return nil, false
	}
	if !validRect(rect) {
		logger.Warn(tag, "无效 rect: %v", rect)
		return nil, false
	}
	raw, err := engine.Scan(rect, mode, returnType)
	if err != nil {
		logger.Warn(tag, "识别异常: %v", err)
		return nil, false
	}
	if raw == "" {
		return nil, false
	}
	result := &Result{Raw: raw, X1: rect.Min.X, Y1: rect.Min.Y}
	switch returnType {
	case ReturnTypeText, ReturnTypeNum:
		result.Text = raw
	default:
		result.Items, result.Text = decode(raw)
	}
	return result, true
}

// Text 识别单行文字（对应 Lua Ocr.text）。
func Text(rect image.Rectangle) string {
	r, ok := Scan(rect, LineMode, ReturnTypeText)
	if !ok {
		return ""
	}
	return r.Text
}

// Number 识别数字（对应 Lua Ocr.number）。
func Number(rect image.Rectangle) (int, bool) {
	r, ok := Scan(rect, LineMode, ReturnTypeNum)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(r.Text))
	if err != nil {
		return 0, false
	}
	return n, true
}

// RecognizeText 多行文本识别（对应 Lua Ocr.recognizeText）。
func RecognizeText(rect image.Rectangle) string {
	r, ok := Scan(rect, MultiMode, ReturnTypeText)
	if !ok {
		return ""
	}
	return r.Text
}

// RecognizeNumber 数字文本识别（对应 Lua Ocr.recognizeNumber）。
func RecognizeNumber(rect image.Rectangle) string {
	r, ok := Scan(rect, MultiMode, ReturnTypeNum)
	if !ok {
		return ""
	}
	return r.Text
}

// RecognizeWords 识别词条列表（对应 Lua Ocr.recognizeWords）。
func RecognizeWords(rect image.Rectangle) []string {
	r, ok := Scan(rect, MultiMode, ReturnTypeJSON)
	if !ok {
		return nil
	}
	var words []string
	for _, item := range r.Items {
		if item.Words != "" {
			words = append(words, item.Words)
		}
	}
	return words
}

var fractionRe = regexp.MustCompile(`(\d+)\s*/\s*(\d+)`)

// ParseFraction 从文本解析 x/x（左=当前，右=上限；对应 Lua Ocr.parseFraction）。
func ParseFraction(text string) (int, int, bool) {
	if text == "" {
		return 0, 0, false
	}
	clean := strings.NewReplacer(" ", "", ",", "", "，", "").Replace(text)
	m := fractionRe.FindStringSubmatch(clean)
	if m == nil {
		return 0, 0, false
	}
	cur, err1 := strconv.Atoi(m[1])
	max, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return cur, max, true
}

// Fraction 识别 x/x 区域（左=当前，右=上限；对应 Lua Ocr.fraction）。
// 失败时返回 raw 文本供调试，ok=false。
func Fraction(rect image.Rectangle) (cur, max int, raw string, ok bool) {
	if !validRect(rect) {
		logger.Warn(tag, "fraction 无效 rect: %v", rect)
		return 0, 0, "", false
	}
	text := Text(rect)
	if text != "" {
		if c, m, ok := ParseFraction(text); ok {
			return c, m, text, true
		}
	}
	// 兜底：以区域中线分左右，分别识数字（slash 漏识时）。
	mid := rect.Min.X + rect.Dx()/2
	left := image.Rect(rect.Min.X, rect.Min.Y, mid, rect.Max.Y)
	right := image.Rect(mid, rect.Min.Y, rect.Max.X, rect.Max.Y)
	c, okC := Number(left)
	m, okM := Number(right)
	if okC && okM {
		return c, m, strconv.Itoa(c) + "/" + strconv.Itoa(m), true
	}
	if text != "" {
		logger.Debug(tag, "fraction 解析失败: %s", text)
	}
	return 0, 0, text, false
}

// Has 区域内是否包含文字（对应 Lua Ocr.has）。
func Has(text string, rect image.Rectangle) bool {
	if text == "" {
		return false
	}
	r, ok := Scan(rect, MultiMode, ReturnTypeJSON)
	if !ok {
		return false
	}
	if strings.Contains(r.Raw, text) || strings.Contains(r.Text, text) {
		return true
	}
	for _, item := range r.Items {
		if strings.Contains(item.Words, text) {
			return true
		}
	}
	return false
}

// Find 识别并返回文字坐标（对应 Lua Ocr.find；引擎 findTapPoint 优先，items 兜底）。
func Find(text string, rect image.Rectangle) (int, int, bool) {
	r, ok := Scan(rect, MultiMode, ReturnTypeJSON)
	if !ok {
		return 0, 0, false
	}
	if engine != nil {
		if x, y, found := engine.FindTapPoint(text, rect); found {
			return x, y, true
		}
	}
	return findInItems(r.Items, text, r.X1, r.Y1)
}

// Tap 识别并点击文字（对应 Lua Ocr.tap）。
func Tap(text string, rect image.Rectangle, delayMs int) bool {
	x, y, ok := Find(text, rect)
	if !ok {
		return false
	}
	touch.TapR(x, y, delayMs)
	return true
}

// Wait 轮询等待文字出现（对应 Lua Ocr.wait）。
func Wait(text string, rect image.Rectangle, timeoutMs, intervalMs int) bool {
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 1000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		if Has(text, rect) {
			return true
		}
		sleep(intervalMs)
	}
	return false
}

// WaitTap 轮询等待文字出现并点击其坐标（对应 Lua Ocr.waitTap）。
func WaitTap(text string, rect image.Rectangle, timeoutMs, intervalMs, delayMs int) (bool, int, int) {
	if text == "" || !validRect(rect) {
		return false, 0, 0
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if intervalMs <= 0 {
		intervalMs = 1000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		if x, y, ok := Find(text, rect); ok {
			touch.TapR(x, y, delayMs)
			return true, x, y
		}
		sleep(intervalMs)
	}
	return false, 0, 0
}
