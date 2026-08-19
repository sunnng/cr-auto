// Package game 对应 Lua 工程的 game/ 目录：任务构建器与业务注册。
package game

import (
	"fmt"

	"app/internal/config"
	"app/internal/core"
	"app/internal/game/arena"
	"app/internal/game/biscuit"
	"app/internal/game/mine/battle"
	"app/internal/game/mine/jelly"
	"app/internal/game/mine/mining"
	"app/internal/game/mine/survey"
	"app/internal/game/popup"
	"app/internal/game/seaside"
	"app/internal/game/square"
	"app/internal/game/starlight"
	"app/internal/lib/dialog"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/store"
	"app/internal/lib/userconfig"
)

const registerTag = "[Register]"

// RegisterAll 对应 Lua game/register.lua 的 Register.all()：清空后注入守卫与任务。
// M2b：全部业务模块按结构直译注册 —— 矿山（勘查/开采/战斗/解除洋菜冻）、
// 海滩交易所、王国竞技场、梦幻繁星岛、布谷鸟广场、洗脆饼词条。
func RegisterAll(s *core.Scheduler, g *core.Guard) {
	s.Clear()
	g.Clear()

	uc := userconfig.New(store.Default())

	mineCfg := func() config.MineConfig {
		var cfg config.MineConfig
		if err := uc.Get("mine", &cfg); err != nil {
			logger.Warn(registerTag, "读取 mine 配置失败: %v", err)
			return config.Static.User.Mine
		}
		return cfg
	}

	arenaCfg := func() config.ArenaConfig {
		var cfg config.ArenaConfig
		if err := uc.Get("arena", &cfg); err != nil {
			logger.Warn(registerTag, "读取 arena 配置失败: %v", err)
			return config.Static.User.Arena
		}
		return cfg
	}

	// ========== 守卫注册（优先级高->低）==========
	unstable := dialog.New(dialog.Def{
		Name:       "网络联机状态不稳定",
		Feature:    popup.UnstableNetworkDef().Feature,
		ConfirmBtn: popup.UnstableNetworkDef().ConfirmBtn,
	}, registerTag)
	g.Register("网络联机状态不稳定", unstable.Def.Feature,
		unstable.ToGuardHandler(dialog.HandleOpts{Action: "confirm", WaitGoneMs: 2000}), 10)
	RegisterSafetyGuards(g, popup.ResourceSpendDef(), popup.SensitivePageDef())

	// 通用离开广场逻辑（对应 Lua task-builder leaveSquareIfNeeded）：非广场上下文直接放行。
	leaveSquare := func() bool {
		if square.IsSquareContext() {
			logger.Info(registerTag, "离开广场")
			if !square.LeaveForOtherTask() {
				logger.Warn(registerTag, "离开广场失败")
				return false
			}
		}
		return true
	}

	// ========== 调度任务注册（矿山优先，空闲时其余模块短切片）==========
	NewTask(s, uc, "矿山勘查", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().SurveyEnabled },
		CheckReady: func() (bool, int) {
			return survey.CheckFarWait()
		},
		WaitHud:     func(remain int) string { return fmt.Sprintf("远距等待 %ds", remain) },
		OnNotReady:  func(int) { updateMineWaitHud("调度等待") },
		LeaveSquare: leaveSquare,
		Action:      func() error { return survey.Run(g) },
	})

	NewTask(s, uc, "矿山开采", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().MiningEnabled },
		CanResume:    func() bool { return mining.IsMiningPage() || mining.IsRewardPage() },
		CheckReady:   func() (bool, int) { return mining.CheckReady() },
		WaitHud:      func(remain int) string { return fmt.Sprintf("busy 等待 %ds", remain) },
		OnNotReady:   func(int) { updateMineWaitHud("调度等待") },
		LeaveSquare:  leaveSquare,
		Action:       func() error { return mining.Run(g) },
	})

	NewTask(s, uc, "矿山战斗", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().BattleEnabled },
		CheckReady: func() (bool, int) {
			interval := mineCfg().BattleIntervalSec
			if interval <= 0 {
				interval = 21600
			}
			remain := battle.GetTimeUntilNext(interval)
			if remain > 0 {
				return false, remain
			}
			return true, 0
		},
		WaitHud:     func(remain int) string { return fmt.Sprintf("冷却等待 %ds", remain) },
		LeaveSquare: leaveSquare,
		Action:      func() error { return battle.Run(g) },
	})

	NewTask(s, uc, "解除洋菜冻", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().JellyEnabled },
		CheckReady:   func() (bool, int) { return jelly.CheckReady() },
		WaitHud:      func(remain int) string { return fmt.Sprintf("冷却等待 %ds", remain) },
		LeaveSquare:  leaveSquare,
		Action:       func() error { return jelly.Run(g) },
	})

	NewTask(s, uc, "海滩交易所", TaskOptions{
		ConfigKey: "seasideMarket",
		CheckReady: func() (bool, int) {
			return seaside.CheckReady()
		},
		WaitHud:            func(remain int) string { return fmt.Sprintf("补货等待 %ds", remain) },
		Precondition:       isMineSchedulerIdle,
		OnPreconditionFail: func() { updateMineWaitHud("矿山待执行") },
		LeaveSquare:        leaveSquare,
		Action:             func() error { return seaside.Run(g) },
	})

	NewTask(s, uc, "王国竞技场", TaskOptions{
		ConfigKey: "arena",
		CheckReady: func() (bool, int) {
			cfg := arenaCfg()
			if arena.IsReachMaxBattles(cfg, arena.Get()) {
				return false, 0
			}
			refreshRemain := arena.GetTimeUntilRefresh()
			if refreshRemain > 0 {
				return false, refreshRemain
			}
			return true, 0
		},
		WaitHud:            func(remain int) string { return fmt.Sprintf("免费刷新等待 %ds", remain) },
		Precondition:       isMineSchedulerIdle,
		OnPreconditionFail: func() { updateMineWaitHud("矿山待执行") },
		LeaveSquare:        leaveSquare,
		Action:             func() error { return arena.Run(g) },
	})

	NewTask(s, uc, "梦幻繁星岛", TaskOptions{
		ConfigKey: "starlight",
		CheckReady: func() (bool, int) {
			if starlight.IsDoneToday() {
				return false, 0
			}
			return true, 0
		},
		Precondition:       isMineSchedulerIdle,
		OnPreconditionFail: func() { updateMineWaitHud("矿山待执行") },
		LeaveSquare:        leaveSquare,
		Action:             func() error { return starlight.Run(g) },
	})

	NewTask(s, uc, "布谷鸟广场", TaskOptions{
		ConfigKey: "square",
		CheckReady: func() (bool, int) {
			if square.IsDoneToday() {
				return false, 0
			}
			return true, 0
		},
		Precondition:       isMineSchedulerIdle,
		OnPreconditionFail: func() { updateMineWaitHud("矿山待执行") },
		Action:             func() error { return square.Run(g) },
	})

	NewTask(s, uc, "洗脆饼词条", TaskOptions{
		ConfigKey:   "biscuit",
		LeaveSquare: leaveSquare,
		Action:      func() error { return biscuit.Run(g) },
	})

	// ========== idle provider 注册（供 Runtime 计算空闲等待）==========
	s.AddIdleProvider("矿山勘查", func() (int, string) {
		if !mineCfg().SurveyEnabled {
			return 0, ""
		}
		remain := survey.RestoreProgress()
		if remain > 0 {
			return remain, fmt.Sprintf("勘查 %ds", remain)
		}
		return 0, ""
	})

	s.AddIdleProvider("矿山开采", func() (int, string) {
		if !mineCfg().MiningEnabled {
			return 0, ""
		}
		remain := mining.RestoreProgress()
		if remain > 0 {
			return remain, fmt.Sprintf("开采 %ds", remain)
		}
		return 0, ""
	})

	s.AddIdleProvider("矿山战斗", func() (int, string) {
		if !mineCfg().BattleEnabled {
			return 0, ""
		}
		remain := battle.GetTimeUntilNext(mineCfg().BattleIntervalSec)
		if remain > 0 {
			return remain, fmt.Sprintf("战斗 %ds", remain)
		}
		return 0, ""
	})

	s.AddIdleProvider("海滩交易所", func() (int, string) {
		var cfg config.SeasideMarketConfig
		if err := uc.Get("seasideMarket", &cfg); err != nil || !cfg.Enabled {
			return 0, ""
		}
		remain := seaside.RestoreProgress()
		if remain > 0 {
			return remain, fmt.Sprintf("海滩 %ds", remain)
		}
		return 0, ""
	})

	s.AddIdleProvider("王国竞技场", func() (int, string) {
		cfg := arenaCfg()
		if !cfg.Enabled {
			return 0, ""
		}
		remain := arena.GetTimeUntilRefresh()
		if remain > 0 {
			return remain, fmt.Sprintf("竞技场 %ds %s", remain, arena.HudText(cfg, arena.Get()))
		}
		return 0, ""
	})

	logger.Info(registerTag, "注入完成 | 守卫 %d 个 任务 %d 个", g.TrapCount(), s.Count())
}

// mineCfgOrDefault 读取矿山配置，失败时回退默认值。
func mineCfgOrDefault() config.MineConfig {
	if cfg, err := userconfig.Mine(); err == nil {
		return cfg
	}
	return config.Static.User.Mine
}

// isMineSchedulerIdle 矿山调度侧是否没有到期任务（对应 Lua isMineSchedulerIdle；
// 海滩交易所/竞技场/繁星岛/布谷鸟广场 等任务的 precondition）。
func isMineSchedulerIdle() bool {
	mineCfg := mineCfgOrDefault()
	if mineCfg.SurveyEnabled {
		if can, _ := survey.CheckFarWait(); can {
			return false
		}
	}
	if mineCfg.MiningEnabled {
		if can, _ := mining.CheckReady(); can {
			return false
		}
	}
	if mineCfg.JellyEnabled {
		if can, _ := jelly.CheckReady(); can {
			return false
		}
	}
	return true
}

// updateMineWaitHud 勘查远距 / 开采 busy 合并展示（对应 Lua updateMineWaitHud）。
func updateMineWaitHud(extra string) {
	mineCfg := mineCfgOrDefault()
	surveySec := 0
	if mineCfg.SurveyEnabled {
		surveySec = survey.RestoreProgress()
	}
	miningSec := 0
	if mineCfg.MiningEnabled {
		miningSec = mining.RestoreProgress()
	}
	if surveySec > 0 || miningSec > 0 {
		status.SetMineWait(status.MineWait{
			SurveySec: surveySec,
			MiningSec: miningSec,
			Extra:     extra,
		})
	}
}
