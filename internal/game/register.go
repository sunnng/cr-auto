// Package game 对应 Lua 工程的 game/ 目录：任务构建器与业务注册。
package game

import (
	"fmt"

	"app/internal/config"
	"app/internal/core"
	"app/internal/game/mine/battle"
	"app/internal/game/mine/jelly"
	"app/internal/game/mine/mining"
	"app/internal/game/mine/survey"
	"app/internal/game/popup"
	"app/internal/lib/dialog"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/store"
	"app/internal/lib/userconfig"
)

const registerTag = "[Register]"

// RegisterAll 对应 Lua game/register.lua 的 Register.all()：清空后注入守卫与任务。
// M2a：网络联机状态不稳定守卫 + 矿山模块（勘查/开采/战斗/解除洋菜冻）按结构直译注册；
// 广场/交易所/竞技场/繁星岛/洗脆饼 等模块在 M2b 起迁移。
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

	// ========== 守卫注册（优先级高->低）==========
	unstable := dialog.New(dialog.Def{
		Name:       "网络联机状态不稳定",
		Feature:    popup.UnstableNetworkDef().Feature,
		ConfirmBtn: popup.UnstableNetworkDef().ConfirmBtn,
	}, registerTag)
	g.Register("网络联机状态不稳定", unstable.Def.Feature,
		unstable.ToGuardHandler(dialog.HandleOpts{Action: "confirm", WaitGoneMs: 2000}), 10)

	// ========== 调度任务注册（矿山优先）==========
	NewTask(s, uc, "矿山勘查", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().SurveyEnabled },
		CheckReady: func() (bool, int) {
			return survey.CheckFarWait()
		},
		WaitHud:     func(remain int) string { return fmt.Sprintf("远距等待 %ds", remain) },
		OnNotReady:  func(int) { updateMineWaitHud("调度等待") },
		LeaveSquare: nil, // M2b 广场模块接入后由 register 注入
		Action:      func() error { return survey.Run(g) },
	})

	NewTask(s, uc, "矿山开采", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().MiningEnabled },
		CanResume:    func() bool { return mining.IsMiningPage() || mining.IsRewardPage() },
		CheckReady:   func() (bool, int) { return mining.CheckReady() },
		WaitHud:      func(remain int) string { return fmt.Sprintf("busy 等待 %ds", remain) },
		OnNotReady:   func(int) { updateMineWaitHud("调度等待") },
		LeaveSquare:  nil,
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
		LeaveSquare: nil,
		Action:      func() error { return battle.Run(g) },
	})

	NewTask(s, uc, "解除洋菜冻", TaskOptions{
		CheckEnabled: func() bool { return mineCfg().JellyEnabled },
		CheckReady:   func() (bool, int) { return jelly.CheckReady() },
		WaitHud:      func(remain int) string { return fmt.Sprintf("冷却等待 %ds", remain) },
		LeaveSquare:  nil,
		Action:       func() error { return jelly.Run(g) },
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
// M2b 交易所/竞技场/繁星岛等任务接入后使用）。
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
