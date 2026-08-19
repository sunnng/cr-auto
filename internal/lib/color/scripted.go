package color

import (
	"image"
	"sync"

	"app/internal/vision"
)

type cmpKey struct {
	x, y int
	spec string
}

// ScriptedScreen 桌面测试用图色桩：按色串/坐标登记命中，不扫描像素。
type ScriptedScreen struct {
	mu        sync.Mutex
	detects   map[string]bool
	cmp       map[cmpKey]bool
	finds     map[string][]image.Point
	DetectsFn func(colors string, sim float32) bool
}

// HitFeatures 登记若干特征为 DetectsMultiColors 与逐点 CmpColor 命中。
func HitFeatures(features ...vision.Feature) *ScriptedScreen {
	s := NewScriptedScreen()
	for _, f := range features {
		s.Hit(f)
	}
	return s
}

// NewScriptedScreen 空桩（一律不命中，除非随后 Hit / FindAt / DetectsFn）。
func NewScriptedScreen() *ScriptedScreen {
	return &ScriptedScreen{
		detects: make(map[string]bool),
		cmp:     make(map[cmpKey]bool),
		finds:   make(map[string][]image.Point),
	}
}

func (s *ScriptedScreen) Begin() {}
func (s *ScriptedScreen) End()   {}

func (s *ScriptedScreen) ensure() {
	if s.detects == nil {
		s.detects = make(map[string]bool)
	}
	if s.cmp == nil {
		s.cmp = make(map[cmpKey]bool)
	}
	if s.finds == nil {
		s.finds = make(map[string][]image.Point)
	}
}

// Hit 登记特征的多点比色与逐点比色命中。
func (s *ScriptedScreen) Hit(f vision.Feature) *ScriptedScreen {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	if colors := vision.DetectsColors(f); colors != "" {
		s.detects[colors] = true
	}
	pts, err := vision.ParsePoints(f.Points)
	if err != nil {
		return s
	}
	for _, p := range pts {
		s.cmp[cmpKey{x: p.X, y: p.Y, spec: vision.CmpSpec(p)}] = true
	}
	return s
}

// Unhit 取消特征的 Detects 与逐点登记。
func (s *ScriptedScreen) Unhit(f vision.Feature) *ScriptedScreen {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	delete(s.detects, vision.DetectsColors(f))
	pts, err := vision.ParsePoints(f.Points)
	if err != nil {
		return s
	}
	for _, p := range pts {
		delete(s.cmp, cmpKey{x: p.X, y: p.Y, spec: vision.CmpSpec(p)})
	}
	return s
}

// HitPoint 只登记逐点 CmpColor（诊断部分命中用，不登记 Detects）。
func (s *ScriptedScreen) HitPoint(p vision.Point) *ScriptedScreen {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	s.cmp[cmpKey{x: p.X, y: p.Y, spec: vision.CmpSpec(p)}] = true
	return s
}

// FindAt 登记找色色串的命中坐标。
func (s *ScriptedScreen) FindAt(def vision.FindDef, pts ...image.Point) *ScriptedScreen {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	colors := vision.FindColors(def)
	if colors == "" {
		return s
	}
	s.finds[colors] = append([]image.Point(nil), pts...)
	return s
}

func (s *ScriptedScreen) DetectsMultiColors(colors string, sim float32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.DetectsFn != nil {
		return s.DetectsFn(colors, sim)
	}
	return s.detects[colors]
}

func (s *ScriptedScreen) CmpColor(x, y int, colorStr string, sim float32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cmp[cmpKey{x: x, y: y, spec: colorStr}]
}

func (s *ScriptedScreen) FindMultiColors(x1, y1, x2, y2 int, colors string, sim float32, dir int) (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pts := s.finds[colors]
	if len(pts) == 0 {
		return -1, -1
	}
	return pts[0].X, pts[0].Y
}

func (s *ScriptedScreen) FindMultiColorsAll(x1, y1, x2, y2 int, colors string, sim float32, dir int) []image.Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	pts := s.finds[colors]
	if len(pts) == 0 {
		return nil
	}
	out := make([]image.Point, len(pts))
	copy(out, pts)
	return out
}
