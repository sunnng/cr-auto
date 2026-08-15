package game

import (
	"testing"

	"app/internal/core"
)

// TestRegisterAllTaskInventory M2b 全量任务与 idle provider 接线（对应 Lua register.all）。
func TestRegisterAllTaskInventory(t *testing.T) {
	setupRegisterTest(t)

	s := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(s, g)

	wantTasks := []string{
		"矿山勘查", "矿山开采", "矿山战斗", "解除洋菜冻",
		"海滩交易所", "王国竞技场", "梦幻繁星岛", "布谷鸟广场", "洗脆饼词条",
	}
	if s.Count() != len(wantTasks) {
		t.Fatalf("task count=%d want %d", s.Count(), len(wantTasks))
	}
	// 注册顺序与 Lua register.lua 一致。
	names := s.Names()
	for i, name := range wantTasks {
		if got := names[i]; got != name {
			t.Fatalf("task[%d]=%q want %q", i, got, name)
		}
	}

	providers := s.GetIdleProviders()
	for _, name := range []string{"矿山勘查", "矿山开采", "矿山战斗", "海滩交易所", "王国竞技场"} {
		if _, ok := providers[name]; !ok {
			t.Fatalf("idle provider %q missing", name)
		}
	}
}

// TestRegisterAllIdleProviders 空闲提供者按配置开关跳过。
func TestRegisterAllIdleProviders(t *testing.T) {
	setupRegisterTest(t)

	// 默认配置：矿山勘查 enabled、海滩交易所/竞技场 disabled。
	s := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(s, g)

	providers := s.GetIdleProviders()

	// 勘查默认开启但无远距等待记录 → 0。
	if remain, _ := providers["矿山勘查"](); remain != 0 {
		t.Fatalf("survey idle remain=%d want 0", remain)
	}
	// 海滩交易所默认关闭 → 0。
	if remain, _ := providers["海滩交易所"](); remain != 0 {
		t.Fatalf("seaside idle remain=%d want 0 (disabled)", remain)
	}
	// 竞技场默认关闭 → 0。
	if remain, _ := providers["王国竞技场"](); remain != 0 {
		t.Fatalf("arena idle remain=%d want 0 (disabled)", remain)
	}
}

// TestRegisterAllGuardTrapCount 守卫注册数量与 Lua 一致。
func TestRegisterAllGuardTrapCount(t *testing.T) {
	setupRegisterTest(t)
	s := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(s, g)
	if g.TrapCount() != 1 {
		t.Fatalf("guard must have 1 trap, got %d", g.TrapCount())
	}
}
