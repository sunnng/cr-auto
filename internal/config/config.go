// Package config 对应 Lua 工程的 config.lua：打包常量的唯一数据源。
package config

// Display 屏幕尺寸固定 1600×900。
type Display struct {
	Width  int
	Height int
}

// Runtime 主循环运行时配置（core.runtime）。
type Runtime struct {
	GuardIntervalMS int // 主线程守卫分片间隔（wait/sleep/状态机轮询）
	GuardSleepMS    int // 保留字段（旧守卫线程用，现未使用）
	StopOnError     bool
	StepDelayMS     int // 有任务时轮间间隔
	IdleDelayMS     int // 没任务时挂机间隔
}

// MineConfig 矿山模块配置段落。
type MineConfig struct {
	SurveyEnabled     bool
	MiningEnabled     bool
	TargetFloor       int
	FarGap            int
	OcrPollSec        int
	FarWaitSec        int
	MiningIntervalSec int
	MiningOreCards    []string
	BattleEnabled     bool
	BattleIntervalSec int
	BattleSoulStones  []string
	JellyEnabled      bool
	JellyIntervalSec  int
}

// BiscuitTarget 洗脆饼目标词条规则。
type BiscuitTarget struct {
	Enabled    bool
	Name       string
	MinPercent int
}

// BiscuitSumRule 洗脆饼总和规则（如 2 条攻击力 ≥ 11）。
type BiscuitSumRule struct {
	Enabled bool
	Name    string
	Count   int
	MinSum  int
}

// BiscuitConfig 洗脆饼配置段落。
type BiscuitConfig struct {
	Enabled  bool
	MaxRolls int
	Targets  []BiscuitTarget
	SumRules []BiscuitSumRule
}

// SquareConfig 布谷鸟广场配置段落。
type SquareConfig struct {
	Enabled          bool
	DailyCap         int
	CheckIntervalSec int
	ChunkSec         int
}

// SeasideMarketConfig 海滩交易所配置段落。
type SeasideMarketConfig struct {
	Enabled          bool
	Items            []string
	RestockBufferSec int
}

// ArenaConfig 王国竞技场配置段落。
type ArenaConfig struct {
	Enabled      bool
	MaxBattles   int
	AutoBuyCount int
	TrophyDiff   int
}

// StarlightConfig 梦幻繁星岛配置段落。
type StarlightConfig struct {
	Enabled bool
}

// User 用户配置默认值（运行时结构，持久化覆盖见 internal/lib/userconfig）。
type User struct {
	Mine          MineConfig          `json:"mine"`
	Biscuit       BiscuitConfig       `json:"biscuit"`
	Square        SquareConfig        `json:"square"`
	SeasideMarket SeasideMarketConfig `json:"seasideMarket"`
	Arena         ArenaConfig         `json:"arena"`
	Starlight     StarlightConfig     `json:"starlight"`
}

// Static 打包常量。
var Static = struct {
	Display Display
	Runtime Runtime
	User    User
}{
	Display: Display{Width: 1600, Height: 900},
	Runtime: Runtime{
		GuardIntervalMS: 500,
		GuardSleepMS:    1000,
		StopOnError:     false,
		StepDelayMS:     5000,
		IdleDelayMS:     30000,
	},
	User: User{
		Mine: MineConfig{
			SurveyEnabled:     true,
			MiningEnabled:     false,
			TargetFloor:       6,
			FarGap:            2,
			OcrPollSec:        60,
			FarWaitSec:        600,
			MiningIntervalSec: 1200,
			MiningOreCards:    []string{"butterAmber", "amberFossil", "sugarOre", "purpleFossil", "emeraldFossil", "flourStone"},
			BattleEnabled:     false,
			BattleIntervalSec: 21600,
			BattleSoulStones:  []string{"妖精王", "莓果", "雷神武将"},
			JellyEnabled:      false,
			JellyIntervalSec:  3600,
		},
		Biscuit: BiscuitConfig{
			Enabled:  false,
			MaxRolls: 500,
			Targets: []BiscuitTarget{
				{Enabled: true, Name: "冷却时间", MinPercent: 5},
				{Enabled: true, Name: "会心", MinPercent: 6},
				{Enabled: false, Name: "", MinPercent: 0},
				{Enabled: false, Name: "", MinPercent: 0},
			},
			SumRules: []BiscuitSumRule{
				{Enabled: true, Name: "攻击力", Count: 2, MinSum: 11},
			},
		},
		Square: SquareConfig{
			Enabled:          true,
			DailyCap:         240,
			CheckIntervalSec: 60,
			ChunkSec:         10,
		},
		SeasideMarket: SeasideMarketConfig{
			Enabled: false,
			Items: []string{
				"灿烂的光之碎片",
				"十分钟加速券",
				"商品1_金紫",
				"商品2_蓝盒",
				"商品3_罗盘",
				"商品4_绿书",
				"商品5_卷轴",
			},
			RestockBufferSec: 30,
		},
		Arena: ArenaConfig{
			Enabled:      false,
			MaxBattles:   0, // 0 表示不限次数
			AutoBuyCount: 0,
			TrophyDiff:   0,
		},
		Starlight: StarlightConfig{Enabled: false},
	},
}
