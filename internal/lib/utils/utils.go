// Package utils 对应 Lua 工程的 lib/utils.lua：字符串/数值解析与坐标换算工具。
package utils

import (
	"image"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ParseNumber 从文本提取纯数字（对应 U.parseNumber）。
// 无法解析时返回 ok=false。
func ParseNumber(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	clean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, text)
	if clean == "" {
		return 0, false
	}
	n, err := strconv.Atoi(clean)
	if err != nil {
		return 0, false
	}
	return n, true
}

var staminaRe = regexp.MustCompile(`([0-9%,.]+)/([0-9%,.]+)`)

// Stamina 体力解析结果（对应 U.parseStamina 返回表）。
type Stamina struct {
	Current, Max int
	Raw          string
}

// ParseStamina 从文本解析 x/y 体力（对应 U.parseStamina）。
// 无法解析时返回 ok=false。
func ParseStamina(text string) (Stamina, bool) {
	if text == "" {
		return Stamina{}, false
	}
	compact := strings.ReplaceAll(text, " ", "")
	m := staminaRe.FindStringSubmatch(compact)
	if m == nil {
		return Stamina{}, false
	}
	current, ok1 := cleanNum(m[1])
	max, ok2 := cleanNum(m[2])
	if !ok1 || !ok2 {
		return Stamina{}, false
	}
	return Stamina{Current: current, Max: max, Raw: text}, true
}

// cleanNum 去掉逗号与小数点后转整数（对应 Lua cleanNum）。
func cleanNum(s string) (int, bool) {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// GenerateNewPos 按基点位移生成新矩形（对应 U.generateNewPos）。
func GenerateNewPos(newBaseX, newBaseY, baseX, baseY int, x1, y1, x2, y2 int) image.Rectangle {
	dx := newBaseX - baseX
	dy := newBaseY - baseY
	return image.Rect(x1+dx, y1+dy, x2+dx, y2+dy)
}

// KeepHanAlphaNum 保留中英数字符、去掉其余字符（对应 U.keepHanAlphaNum；
// Lua 模式 [^%w%u%l一-龥] 即 ASCII 字母数字 + 汉字 U+4E00-U+9FA5）。
func KeepHanAlphaNum(str string) string {
	if str == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range str {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || unicode.Is(unicode.Han, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
