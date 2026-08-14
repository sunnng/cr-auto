// Package vision 提供不依赖 AutoGo 实时屏幕 API 的帧比色识别：
// 多点比色（特征库）、找色（findMultiColor）都在一张 *image.NRGBA 帧上由纯 Go 算法完成。
// 色点参数格式（"x|y|rrggbb-偏色" + 相似度）与图色工具产出、Lua 特征库、AutoGo API 一致。
package vision

import (
	"fmt"
	"image"
	"math"
	"sort"
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

func parsePoints(spec string) ([]Point, error) {
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

// parseOffsetTolerance 解析偏色后缀："101010" 表示 R/G/B 各 ±0x10，"10" 表示 ±0x10。
// 解析失败时回退默认容差。
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

// requiredPoints 返回相似度 sim 要求的最少命中色点数。
// sim 缺省（<=0）视为 1：全部色点命中。
func requiredPoints(sim float32, total int) int {
	if sim <= 0 || sim >= 1 || total <= 0 {
		return total
	}
	return int(math.Ceil(float64(sim) * float64(total)))
}

func channelNear(actual, expected, tolerance uint8) bool {
	delta := int(actual) - int(expected)
	if delta < 0 {
		delta = -delta
	}
	return delta <= int(tolerance)
}

func pointNear(img *image.NRGBA, p Point) bool {
	bounds := img.Bounds()
	if p.X < bounds.Min.X || p.X >= bounds.Max.X || p.Y < bounds.Min.Y || p.Y >= bounds.Max.Y {
		return false
	}
	offset := img.PixOffset(p.X, p.Y)
	return channelNear(img.Pix[offset], p.R, p.Tol) &&
		channelNear(img.Pix[offset+1], p.G, p.Tol) &&
		channelNear(img.Pix[offset+2], p.B, p.Tol)
}

// Match 单特征比色是否匹配：命中色点数达到相似度要求即视为匹配。
func Match(img *image.NRGBA, f Feature) bool {
	if img == nil {
		return false
	}
	points, err := parsePoints(f.Points)
	if err != nil {
		return false
	}
	matched := 0
	for _, p := range points {
		if pointNear(img, p) {
			matched++
		}
	}
	return matched >= requiredPoints(f.Sim, len(points))
}

// MatchAny 多个特征任一匹配，返回命中的下标（0 起），未命中返回 -1。
func MatchAny(img *image.NRGBA, features []Feature) (bool, int) {
	for i, f := range features {
		if Match(img, f) {
			return true, i
		}
	}
	return false, -1
}

// ColorSpec 单个颜色规格："rrggbb[-偏色]"。
type ColorSpec struct {
	R, G, B uint8
	Tol     uint8
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

func colorSpecNear(img *image.NRGBA, x, y int, spec ColorSpec) bool {
	bounds := img.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return false
	}
	offset := img.PixOffset(x, y)
	return channelNear(img.Pix[offset], spec.R, spec.Tol) &&
		channelNear(img.Pix[offset+1], spec.G, spec.Tol) &&
		channelNear(img.Pix[offset+2], spec.B, spec.Tol)
}

func anyColorSpecNear(img *image.NRGBA, x, y int, specs []ColorSpec) bool {
	for _, spec := range specs {
		if colorSpecNear(img, x, y, spec) {
			return true
		}
	}
	return false
}

// FindDef 区域内找色定义 {x1,y1,x2,y2, firstColor, offsetColors, dir, sim}。
// OffsetColors 为 "dx,dy,rrggbb-偏色,dx,dy,rrggbb-偏色,..." 三元组序列。
type FindDef struct {
	Region       image.Rectangle
	FirstColor   string
	OffsetColors string
	Dir          int
	Sim          float32
}

// FindMultiColor 在区域内查找匹配的多点颜色序列，返回首个命中点的坐标。
func FindMultiColor(img *image.NRGBA, def FindDef) (x, y int, ok bool) {
	if img == nil {
		return 0, 0, false
	}
	anchors := findMultiColor(img, def)
	if len(anchors) == 0 {
		return 0, 0, false
	}
	first := anchors[0]
	return first.X, first.Y, true
}

// FindMultiColorAll 在区域内查找全部命中点，按扫描方向排序。
func FindMultiColorAll(img *image.NRGBA, def FindDef) []image.Point {
	if img == nil {
		return nil
	}
	return findMultiColor(img, def)
}

type findPlan struct {
	first []ColorSpec
	refs  []offsetSpec
}

// offsetSpec 找色定义中的一条相对色点：偏移量 + 候选颜色（"|" 分隔可多选）。
type offsetSpec struct {
	DX, DY int
	colors []ColorSpec
}

func buildFindPlan(def FindDef) (findPlan, error) {
	var plan findPlan
	plan.first = ParseColorSpec(def.FirstColor)
	if len(plan.first) == 0 {
		return plan, fmt.Errorf("vision: 找色定义缺少首色: %q", def.FirstColor)
	}
	if def.OffsetColors == "" {
		return plan, nil
	}
	offsets := strings.Split(def.OffsetColors, ",")
	if len(offsets)%3 != 0 {
		return plan, fmt.Errorf("vision: offsetColors 应为 dx,dy,color 三元组: %q", def.OffsetColors)
	}
	for i := 0; i < len(offsets); i += 3 {
		dx, err1 := strconv.Atoi(strings.TrimSpace(offsets[i]))
		dy, err2 := strconv.Atoi(strings.TrimSpace(offsets[i+1]))
		if err1 != nil || err2 != nil {
			return plan, fmt.Errorf("vision: offset 坐标非法: %q", def.OffsetColors)
		}
		specs := ParseColorSpec(offsets[i+2])
		if len(specs) == 0 {
			return plan, fmt.Errorf("vision: offset 颜色非法: %q", offsets[i+2])
		}
		plan.refs = append(plan.refs, offsetSpec{DX: dx, DY: dy, colors: specs})
	}
	return plan, nil
}

func findMultiColor(img *image.NRGBA, def FindDef) []image.Point {
	plan, err := buildFindPlan(def)
	if err != nil {
		return nil
	}
	region := def.Region.Intersect(img.Bounds())
	if region.Empty() {
		return nil
	}
	sim := def.Sim
	if sim <= 0 {
		sim = 1
	}

	seq := make([]image.Point, 0, region.Dx()*region.Dy())
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			seq = append(seq, image.Point{X: x, Y: y})
		}
	}
	orderScanByDir(seq, def.Dir)

	anchors := make([]image.Point, 0)
	for _, c := range seq {
		if !anyColorSpecNear(img, c.X, c.Y, plan.first) {
			continue
		}
		matched := 0
		for _, ref := range plan.refs {
			if anyColorSpecNear(img, c.X+ref.DX, c.Y+ref.DY, ref.colors) {
				matched++
			}
		}
		total := len(plan.refs)
		if total == 0 || matched >= requiredPoints(sim, total) {
			anchors = append(anchors, image.Point{X: c.X, Y: c.Y})
		}
	}
	return anchors
}

// orderScanByDir 按 dir 调整扫描顺序：0 左上→右下，1 右上→左下，2 左下→右上，3 右下→左上。
func orderScanByDir(seq []image.Point, dir int) {
	if len(seq) <= 1 || dir == 0 {
		return
	}
	flipY := dir == 2 || dir == 3
	flipX := dir == 1 || dir == 3
	for i := range seq {
		if flipX {
			seq[i].X = -seq[i].X
		}
		if flipY {
			seq[i].Y = -seq[i].Y
		}
	}
	sort.Slice(seq, func(i, j int) bool {
		if seq[i].Y != seq[j].Y {
			return seq[i].Y < seq[j].Y
		}
		return seq[i].X < seq[j].X
	})
	for i := range seq {
		if flipX {
			seq[i].X = -seq[i].X
		}
		if flipY {
			seq[i].Y = -seq[i].Y
		}
	}
}
