// Package arena 对应 Lua 工程的 game/常规_王国竞技场/：竞技场特征库、页面、会话、路由与任务。
package arena

import (
	"errors"
	"fmt"
	"time"

	"app/internal/config"
	"app/internal/core"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/touch"
	"app/internal/lib/userconfig"
)

const taskTag = "[王国竞技场.任务]"

func arenaCfg() config.ArenaConfig {
	cfg, err := userconfig.Arena()
	if err != nil {
		logger.Warn(taskTag, "读取竞技场配置失败: %v", err)
		return config.Static.User.Arena
	}
	return cfg
}

func syncSession() {
	d := Get()
	if medal, ticket, mOk, tOk := ReadMedalAndTicket(); mOk || tOk {
		if mOk {
			d.Medals = medal
		}
		if tOk {
			d.Tickets = ticket
		}
		Set(d)
	}
	if trophies, ok := ReadTrophyCount(); ok {
		d.Trophies = trophies
		Set(d)
	}
}

func syncStatus(cfg config.ArenaConfig) {
	syncSession()
	status.SetTask("王国竞技场", HudText(cfg, Get()))
}

func isReachMaxBattles(cfg config.ArenaConfig) bool {
	return IsReachMaxBattles(cfg, Get())
}

func tryAutoBuy(cfg config.ArenaConfig) bool {
	limit := cfg.AutoBuyCount
	d := Get()
	bought := d.BuyCount

	if limit <= 0 {
		logger.Info(taskTag, " 自动买票未启用，离开竞技场")
		return false
	}
	if bought >= limit {
		logger.Info(taskTag, " 已达买票上限")
		return false
	}

	logger.Info(taskTag, " 自动买票 第%d/%d次", bought+1, limit)
	BuyTicket()
	Update(func(d *ArenaData) { d.BuyCount = bought + 1 })
	color.Sleep(1500, 500)
	syncStatus(cfg)

	if Get().Tickets > 0 {
		return true
	}
	logger.Warn(taskTag, " 买票后门票仍为0，离开竞技场")
	return false
}

func doBattle(info OpponentInfo, cfg config.ArenaConfig) (bool, error) {
	logger.Info(taskTag, " 开战 奖杯=%d", info.Trophies)
	touch.TapR(info.Site.X, info.Site.Y, 500)

	result := RunBattle()
	if result == "" {
		logger.Warn(taskTag, " 战斗失败")
		return false, errors.New("战斗失败")
	}

	Update(func(d *ArenaData) {
		switch result {
		case "胜利":
			d.Wins++
		case "平局":
			d.Draws++
		case "失败":
			d.Losses++
		}
		if d.Tickets > 0 {
			d.Tickets--
		}
	})
	syncStatus(cfg)

	logger.Info(taskTag, " 战斗完成 result=%s %s", result, Describe(Get()))
	return true, nil
}

// selectAndFight 扫描当前页 → 翻页扫描 → 刷新/退出。
// 返回 ("fought"|"refreshed"|"exit")，失败返回 ("", err)。
func selectAndFight(cfg config.ArenaConfig) (string, error) {
	myTrophy := Get().Trophies

	// 第 1 次扫描：当前页。
	if info, ok := FindFirstValidOpponent(cfg, myTrophy); ok {
		if ok, err := doBattle(info, cfg); !ok {
			return "", err
		}
		return "fought", nil
	}

	logger.Info(taskTag, " 当前页无合适敌人，翻页扫描")
	SwipePageLeft()

	// 第 2 次扫描：翻页后。
	if info, ok := FindFirstValidOpponent(cfg, myTrophy); ok {
		if ok, err := doBattle(info, cfg); !ok {
			return "", err
		}
		return "fought", nil
	}

	logger.Info(taskTag, " 翻页后仍无合适敌人")

	// 尝试免费刷新。
	if IsFreeRefresh() {
		logger.Info(taskTag, " 点击免费刷新")
		TapFreeRefresh()
		ClearNextFreeRefresh()
		return "refreshed", nil
	}

	// 不可刷新：解析倒计时并退出。
	seconds, ok := ReadRefreshCountdown()
	if ok && seconds > 0 {
		nextAt := time.Now().Unix() + int64(seconds)
		SetNextFreeRefreshAt(nextAt)
		logger.Info(taskTag, " 免费刷新倒计时 %d秒，下次进入时间 %s", seconds, time.Unix(nextAt, 0).Format("15:04:05"))
	} else {
		logger.Warn(taskTag, " 未识别免费刷新且倒计时解析失败，默认 30 分钟后重试")
		SetNextFreeRefreshAt(time.Now().Unix() + 30*60)
	}

	return "exit", nil
}

// Run 运行王国竞技场任务（对应 Task.run）。
// 返回 nil 表示正常结束，core.ErrSkip 表示任务未启用或已达上限。
func Run(_ *core.Guard) error {
	cfg := arenaCfg()
	if !cfg.Enabled {
		logger.Info(taskTag, " 任务未启用，跳过")
		return core.ErrSkip
	}

	if isReachMaxBattles(cfg) {
		logger.Info(taskTag, " 已达战斗上限")
		return core.ErrSkip
	}

	maxText := "∞"
	if cfg.MaxBattles > 0 {
		maxText = fmt.Sprintf("%d", cfg.MaxBattles)
	}
	logger.Info(taskTag, " 启动 上限=%s 奖杯差阈=%d 自动买票=%d", maxText, cfg.TrophyDiff, cfg.AutoBuyCount)

	status.SetTask("王国竞技场", "进入中…")
	if !Enter() {
		logger.Warn(taskTag, " 进入竞技场失败")
		status.SetTask("王国竞技场", "进入失败")
		return core.ErrSkip
	}

	running := true
	for running {
		syncStatus(cfg)

		if isReachMaxBattles(cfg) {
			logger.Info(taskTag, " 达到战斗上限，退出")
			break
		}

		if Get().Tickets <= 0 {
			if !tryAutoBuy(cfg) {
				break
			}
			syncStatus(cfg)
		}

		result, err := selectAndFight(cfg)
		if result == "" {
			logger.Warn(taskTag, " 任务结束：%v", err)
			status.SetTask("王国竞技场", "失败: "+err.Error())
			running = false
		} else if result == "exit" {
			running = false
		}
		// "refreshed" / "fought" 继续循环。
	}

	syncSession()
	logger.Info(taskTag, " 任务结束 %s", Describe(Get()))
	Leave()
	return nil
}
