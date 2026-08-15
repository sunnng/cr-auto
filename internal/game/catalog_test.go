package game

import (
	"testing"

	"app/internal/core"
	"app/internal/lib/store"
	"app/internal/lib/userconfig"
)

// TestCatalogMatchesRegisterAllOrder 目录元数据必须与 RegisterAll 注册顺序与名称一致，
// 防止任务名在两侧漂移。
func TestCatalogMatchesRegisterAllOrder(t *testing.T) {
	setupRegisterTest(t)

	s := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(s, g)
	names := s.Names()

	catalog := Catalog()
	if len(catalog) != len(names) {
		t.Fatalf("catalog=%d tasks must match register=%d", len(catalog), len(names))
	}
	for i, meta := range catalog {
		if meta.Name != names[i] {
			t.Fatalf("catalog[%d]=%q must match register order %q", i, meta.Name, names[i])
		}
		if meta.ID == "" || meta.Description == "" || meta.Section == "" || meta.Field == "" || meta.MaxRuns < 1 {
			t.Fatalf("catalog[%d] incomplete metadata: %+v", i, meta)
		}
	}
}

// TestCatalogTaskIDsUnique 面板草稿以 ID 为键，目录 ID 必须唯一。
func TestCatalogTaskIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, meta := range Catalog() {
		if seen[meta.ID] {
			t.Fatalf("duplicate task ID %q", meta.ID)
		}
		seen[meta.ID] = true
	}
}

// TestApplyTaskSwitchesWritesUserConfig 面板保存的任务开关写入 userconfig 后，
// 新实例（register 同路径）能读到合并结果；未知 ID 被忽略。
func TestApplyTaskSwitchesWritesUserConfig(t *testing.T) {
	setupRegisterTest(t)
	uc := userconfig.New(store.Default())

	if err := ApplyTaskSwitches(uc, map[string]bool{
		"mine_survey":    false,
		"mine_mining":    true,
		"seaside_market": true,
		"unknown_id":     true,
	}); err != nil {
		t.Fatal(err)
	}

	fresh := userconfig.New(store.Default())
	var mineCfg struct {
		SurveyEnabled bool
		MiningEnabled bool
	}
	if err := fresh.Get("mine", &mineCfg); err != nil {
		t.Fatal(err)
	}
	if mineCfg.SurveyEnabled || !mineCfg.MiningEnabled {
		t.Fatalf("mine switches not applied: %+v", mineCfg)
	}
	var seaside struct{ Enabled bool }
	if err := fresh.Get("seasideMarket", &seaside); err != nil {
		t.Fatal(err)
	}
	if !seaside.Enabled {
		t.Fatalf("seaside switch not applied: %+v", seaside)
	}
}

// TestApplyTaskSwitchesPartialMergeKeepsOtherMineFields 只改 mine 段一个开关时，
// 段内其余字段（默认值或历史保存值）必须保留。
func TestApplyTaskSwitchesPartialMergeKeepsOtherMineFields(t *testing.T) {
	setupRegisterTest(t)
	uc := userconfig.New(store.Default())

	if err := ApplyTaskSwitches(uc, map[string]bool{"mine_mining": true}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyTaskSwitches(uc, map[string]bool{"mine_survey": false}); err != nil {
		t.Fatal(err)
	}

	fresh := userconfig.New(store.Default())
	var mineCfg struct {
		SurveyEnabled bool
		MiningEnabled bool
		BattleEnabled bool
		JellyEnabled  bool
	}
	if err := fresh.Get("mine", &mineCfg); err != nil {
		t.Fatal(err)
	}
	if mineCfg.SurveyEnabled || !mineCfg.MiningEnabled || mineCfg.BattleEnabled || mineCfg.JellyEnabled {
		t.Fatalf("partial mine switch must merge, got %+v", mineCfg)
	}
}

// TestApplyTaskSwitchesConsumedByRegister 目录开关写入后，RegisterAll 注入的任务
// CheckEnabled 必须消费同一份配置（全关 → 全部任务条件不满足）。
func TestApplyTaskSwitchesConsumedByRegister(t *testing.T) {
	setupRegisterTest(t)
	uc := userconfig.New(store.Default())

	switches := map[string]bool{}
	for _, meta := range Catalog() {
		switches[meta.ID] = false
	}
	if err := ApplyTaskSwitches(uc, switches); err != nil {
		t.Fatal(err)
	}

	s := core.NewScheduler()
	g := core.NewGuard()
	RegisterAll(s, g)

	for _, task := range s.Tasks() {
		if task.Condition() {
			t.Fatalf("task %q must be disabled when all switches are off", task.Name)
		}
	}
}

// TestLoadTaskSwitchesReadsBackSavedSwitches 保存 → 重读（新实例）必须一致，
// 未保存过时回读默认值。
func TestLoadTaskSwitchesReadsBackSavedSwitches(t *testing.T) {
	setupRegisterTest(t)
	uc := userconfig.New(store.Default())

	if err := ApplyTaskSwitches(uc, map[string]bool{
		"mine_survey":    false,
		"mine_mining":    true,
		"seaside_market": true,
	}); err != nil {
		t.Fatal(err)
	}

	fresh := userconfig.New(store.Default())
	switches, err := LoadTaskSwitches(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if len(switches) != len(Catalog()) {
		t.Fatalf("switches=%d want %d catalog entries", len(switches), len(Catalog()))
	}
	if switches["mine_survey"] || !switches["mine_mining"] || !switches["seaside_market"] {
		t.Fatalf("saved switches must read back: %+v", switches)
	}
	// 未显式保存的任务回读默认值（config.Static.User：square.Enabled=true）。
	if !switches["square"] {
		t.Fatal("square default must be enabled after merge")
	}
}
