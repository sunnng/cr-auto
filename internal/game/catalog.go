// Package game 对应 Lua 工程的 game/ 目录：任务构建器与业务注册。
package game

import (
	"app/internal/lib/userconfig"
)

// TaskMeta 任务目录元数据：控制面板目录展示与 CommandSave 配置接线的唯一数据源。
// ID 是面板草稿 Tasks 的稳定键；Section/Field 指向 userconfig 中该任务的开关字段
// （register 各任务 CheckEnabled 消费同一份配置）。
// MaxRuns 是该任务“单次上限”的默认值与滑动条上限（面板“单次上限”语义）：
// 调度器每轮最多执行该次数，且连续执行之间重新求值条件（见 core.Scheduler）。
// 矿山/繁星岛为 1（单次流程，安全策略固定）；分片型任务按模块自然配额设置
// （广场 24 片≈每日上限、交易所 5 次≈补货缓冲、竞技场/洗脆饼 100 次）。
type TaskMeta struct {
	ID          string
	Name        string
	Description string
	Section     string
	Field       string
	MaxRuns     int
}

// taskCatalog 全部已注册任务目录，顺序与 RegisterAll 注册顺序一致（见 catalog_test）。
// Section/Field 与 config.Static.User 各配置段的开关字段一一对应。
var taskCatalog = []TaskMeta{
	{
		ID:          "mine_survey",
		Name:        "矿山勘查",
		Description: "启动矿山勘查并进入远距等待；勘查期间调度让渡给其余到期任务",
		Section:     "mine",
		Field:       "SurveyEnabled",
		MaxRuns:     1,
	},
	{
		ID:          "mine_mining",
		Name:        "矿山开采",
		Description: "按白名单矿石卡执行开采流程并结算奖励",
		Section:     "mine",
		Field:       "MiningEnabled",
		MaxRuns:     1,
	},
	{
		ID:          "mine_battle",
		Name:        "矿山战斗",
		Description: "按灵魂石白名单执行矿山战斗并确认结算",
		Section:     "mine",
		Field:       "BattleEnabled",
		MaxRuns:     1,
	},
	{
		ID:          "jelly_release",
		Name:        "解除洋菜冻",
		Description: "执行解除洋菜冻流程并保存冷却间隔",
		Section:     "mine",
		Field:       "JellyEnabled",
		MaxRuns:     1,
	},
	{
		ID:          "seaside_market",
		Name:        "海滩交易所",
		Description: "按白名单商品补货、免费刷新并等待补货间隔",
		Section:     "seasideMarket",
		Field:       "Enabled",
		MaxRuns:     5,
	},
	{
		ID:          "arena",
		Name:        "王国竞技场",
		Description: "按次数与免费刷新预算执行竞技场对战",
		Section:     "arena",
		Field:       "Enabled",
		MaxRuns:     100,
	},
	{
		ID:          "starlight",
		Name:        "梦幻繁星岛",
		Description: "每日进入繁星岛查看小岛并领取可见任务奖励",
		Section:     "starlight",
		Field:       "Enabled",
		MaxRuns:     1,
	},
	{
		ID:          "square",
		Name:        "布谷鸟广场",
		Description: "在广场累计有效停留，达到每日上限后领取奖励并返回王国",
		Section:     "square",
		Field:       "Enabled",
		MaxRuns:     24,
	},
	{
		ID:          "biscuit",
		Name:        "洗脆饼词条",
		Description: "按目标词条与总和规则执行洗脆饼重洗",
		Section:     "biscuit",
		Field:       "Enabled",
		MaxRuns:     100,
	},
}

// Catalog 返回全部已注册任务的目录元数据副本（注册顺序与 register.lua 一致）。
func Catalog() []TaskMeta {
	return append([]TaskMeta(nil), taskCatalog...)
}

// ApplyTaskSwitches 把面板保存的任务开关（任务 ID -> enabled）合并写入 userconfig，
// register 各任务 CheckEnabled 消费同一份配置；未出现在目录中的 ID 被忽略。
// 同一配置段的多个开关按段聚合一次写入，段内其余字段保留。
func ApplyTaskSwitches(uc *userconfig.UserConfig, switches map[string]bool) error {
	patches := make(map[string]map[string]any)
	for _, meta := range taskCatalog {
		value, ok := switches[meta.ID]
		if !ok {
			continue
		}
		if patches[meta.Section] == nil {
			patches[meta.Section] = map[string]any{}
		}
		patches[meta.Section][meta.Field] = value
	}
	for section, patch := range patches {
		if err := uc.Set(section, patch); err != nil {
			return err
		}
	}
	return nil
}

// LoadTaskSwitches 从 userconfig 回读全部任务开关（面板草稿初始回填用），
// 返回目录 ID -> 开关值（默认值 + 保存覆盖合并后的结果）。
func LoadTaskSwitches(uc *userconfig.UserConfig) (map[string]bool, error) {
	switches := make(map[string]bool, len(taskCatalog))
	for _, meta := range taskCatalog {
		var section map[string]any
		if err := uc.Get(meta.Section, &section); err != nil {
			return nil, err
		}
		value, ok := section[meta.Field].(bool)
		if !ok {
			value = false
		}
		switches[meta.ID] = value
	}
	return switches, nil
}
