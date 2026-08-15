// Package biscuit 对应 Lua 工程的 game/功能_洗脆饼/：脆饼词条参考表与洗脆饼任务。
package biscuit

import "fmt"

// Entry 词条参考表项（对应 Lua 词条库.entries 的一项）。
type Entry struct {
	Name     string
	MinValue float64
	MaxValue float64
}

var entries = []Entry{
	{Name: "攻击力", MinValue: 3, MaxValue: 7.5},
	{Name: "防御力", MinValue: 5, MaxValue: 7.5},
	{Name: "生命值", MinValue: 3, MaxValue: 15},
	{Name: "攻击速度", MinValue: 3, MaxValue: 10},
	{Name: "会心", MinValue: 3, MaxValue: 7},
	{Name: "冷却时间", MinValue: 2, MaxValue: 6},
	{Name: "伤害减免", MinValue: 5, MaxValue: 10},
	{Name: "会心伤害减免", MinValue: 4, MaxValue: 10},
	{Name: "增益效果增强", MinValue: 2, MaxValue: 5},
	{Name: "减益效果减免", MinValue: 2, MaxValue: 5},
	{Name: "无视伤害减免", MinValue: 5, MaxValue: 15},
	{Name: "电属性伤害提升", MinValue: 8, MaxValue: 15},
	{Name: "火属性伤害提升", MinValue: 8, MaxValue: 15},
	{Name: "暗属性伤害提升", MinValue: 8, MaxValue: 15},
	{Name: "毒属性伤害提升", MinValue: 8, MaxValue: 15},
}

// Names 全部词条名（对应 M.names）。
func Names() []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, e.Name)
	}
	return list
}

// Find 按名查找词条（对应 M.find）。
func Find(name string) (Entry, bool) {
	for _, e := range entries {
		if e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}

// ValueBounds 词条数值区间（对应 M.valueBounds）。
func ValueBounds(name string) (float64, float64, bool) {
	e, ok := Find(name)
	if !ok {
		return 0, 0, false
	}
	return e.MinValue, e.MaxValue, true
}

// SumBounds 取 count 条时的总和区间（对应 M.sumBounds）。
func SumBounds(name string, count int) (float64, float64, bool) {
	min, max, ok := ValueBounds(name)
	if !ok || count < 1 {
		return 0, 0, false
	}
	if count > 4 {
		count = 4
	}
	return min * float64(count), max * float64(count), true
}

// RangeHint 数值区间提示（对应 M.rangeHint）。
func RangeHint(name string) string {
	e, ok := Find(name)
	if !ok {
		return ""
	}
	return fmt.Sprintf("范围 %g%%~%g%%", e.MinValue, e.MaxValue)
}
