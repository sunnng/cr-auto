// Package vision 保存特征库色点格式与 AutoGo 图色 API 之间的字符串转换。
// 运行时比色/找色由 internal/lib/color 注入的 Screen（设备端为 AutoGo images API）完成。
package vision

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

const defaultTolerance = 0x10

// Point 特征串中的单条色点规格：坐标、期望颜色与逐通道偏色容差。
type Point struct {
	X, Y    int
	R, G, B uint8
	Tol     uint8
}

// Feature 特征库中的单条多点比色规格，即 "x|y|color-偏移" 串加相似度。
// Points 示例："1380|60|f7e5cb-101010,59|323|b3001b-101010"
type Feature struct {
	Points string
	Sim    float32
}

// PointResult 单点比色结果（识别诊断锚点展示用）。
type PointResult struct {
	Point   Point
	Matched bool
}

// ColorSpec 单个颜色规格："rrggbb[-偏色]"。
type ColorSpec struct {
	R, G, B uint8
	Tol     uint8
}

// FindDef 区域内找色定义 {x1,y1,x2,y2, firstColor, offsetColors, dir, sim}。
// OffsetColors 为 "dx,dy,rrggbb-偏色,..." 三元组，或 Lua 管道 "dx|dy|color|..."。
type FindDef struct {
	Region       image.Rectangle
	FirstColor   string
	OffsetColors string
	Dir          int
	Sim          float32
}

// ParsePoints 解析 "x|y|rrggbb-偏色,..." 特征串。
func ParsePoints(spec string) ([]Point, error) {
	var points []Point
	for _, chunk := range strings.Split(spec, ",") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.Split(chunk, "|")
		if len(parts) != 3 {
			return nil, fmt.Errorf("vision: 色点格式应为 x|y|rrggbb-偏色: %q", chunk)
		}
		x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("vision: 色点 x 非法: %q", chunk)
		}
		y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("vision: 色点 y 非法: %q", chunk)
		}
		colorSpec := strings.TrimSpace(parts[2])
		hex := colorSpec
		tol := uint8(defaultTolerance)
		if dash := strings.LastIndex(colorSpec, "-"); dash >= 0 {
			hex = colorSpec[:dash]
			tol = parseOffsetTolerance(colorSpec[dash+1:])
		}
		if len(hex) != 6 {
			return nil, fmt.Errorf("vision: 颜色应为 6 位 hex: %q", chunk)
		}
		rgb, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return nil, fmt.Errorf("vision: 颜色非法: %q", chunk)
		}
		points = append(points, Point{
			X: x, Y: y,
			R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb),
			Tol: tol,
		})
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("vision: 特征串不含色点: %q", spec)
	}
	return points, nil
}

func parseOffsetTolerance(spec string) uint8 {
	channel := spec
	if len(spec) == 6 {
		channel = spec[:2]
	}
	if channel == "" {
		return defaultTolerance
	}
	v, err := strconv.ParseUint(channel, 16, 8)
	if err != nil {
		return defaultTolerance
	}
	return uint8(v)
}

// ParseColorSpec 解析 "FFFFFF|CCCCCC-101010" 风格的颜色规格串（多个候选以 "|" 分隔）。
func ParseColorSpec(spec string) []ColorSpec {
	var out []ColorSpec
	for _, part := range strings.Split(spec, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		hex := part
		tol := uint8(defaultTolerance)
		if dash := strings.LastIndex(part, "-"); dash >= 0 {
			hex = part[:dash]
			tol = parseOffsetTolerance(part[dash+1:])
		}
		if len(hex) != 6 {
			continue
		}
		rgb, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			continue
		}
		out = append(out, ColorSpec{
			R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), Tol: tol,
		})
	}
	return out
}

// DetectsColors 把特征串转成 AutoGo DetectsMultiColors 色串：点内 "|" 改为 ","。
func DetectsColors(f Feature) string {
	spec := strings.TrimSpace(f.Points)
	if spec == "" {
		return ""
	}
	if _, err := ParsePoints(spec); err != nil {
		return ""
	}
	return strings.ReplaceAll(spec, "|", ",")
}

// FindColors 把找色定义转成 AutoGo FindMultiColors 色串："first,dx,dy,color,..."。
// 非法首色或非法 offset 三元组返回空串。
func FindColors(def FindDef) string {
	first := strings.TrimSpace(def.FirstColor)
	if len(ParseColorSpec(first)) == 0 {
		return ""
	}
	offsets := strings.TrimSpace(def.OffsetColors)
	if offsets == "" {
		return first
	}
	parts := splitOffsetTriplets(offsets)
	if len(parts)%3 != 0 {
		return ""
	}
	for i := 0; i < len(parts); i += 3 {
		if _, err := strconv.Atoi(strings.TrimSpace(parts[i])); err != nil {
			return ""
		}
		if _, err := strconv.Atoi(strings.TrimSpace(parts[i+1])); err != nil {
			return ""
		}
		if len(ParseColorSpec(parts[i+2])) == 0 {
			return ""
		}
	}
	joined := make([]string, 0, 1+len(parts))
	joined = append(joined, first)
	for _, p := range parts {
		joined = append(joined, strings.TrimSpace(p))
	}
	return strings.Join(joined, ",")
}

// CmpSpec 把色点转成 AutoGo CmpColor 色串："rrggbb" 或 "rrggbb-tttttt"。
func CmpSpec(p Point) string {
	hex := fmt.Sprintf("%02x%02x%02x", p.R, p.G, p.B)
	if p.Tol == 0 {
		return hex
	}
	return fmt.Sprintf("%s-%02x%02x%02x", hex, p.Tol, p.Tol, p.Tol)
}

// splitOffsetTriplets 把 offsetColors 按三元组分隔拆分：逗号格式优先
// （颜色元素内可用 "|" 多候选），非三元组长度时回退管道格式（Lua 原样）。
func splitOffsetTriplets(spec string) []string {
	parts := strings.Split(spec, ",")
	if len(parts)%3 == 0 {
		return parts
	}
	return strings.Split(spec, "|")
}
