// Package starlight 对应 Lua 工程的 game/常规_梦幻繁星岛/：繁星岛坐标库、页面、会话、路由与任务。
package starlight

import (
	"errors"

	"app/internal/core"
	"app/internal/game/kingdom"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
)

const taskTag = "[梦幻繁星岛任务]"

func updateHud(state string) {
	status.SetTask("梦幻繁星岛", state)
}

var starlightHandlers = map[string]core.StateHandler{
	"check": func(sm *core.StateMachine) (string, error) {
		if IsDoneToday() {
			logger.Info(taskTag, " 今日已执行，跳过")
			return core.DONE, nil
		}
		return "detect", nil
	},

	"detect": func(sm *core.StateMachine) (string, error) {
		switch {
		case IsHomePage():
			logger.Info(taskTag, " [detect] 当前在梦幻繁星岛首页")
			return "openManual", nil
		case IsManualPage():
			logger.Info(taskTag, " [detect] 当前在航海手册页")
			return "enterIsland", nil
		case IsVanillaIslandPage():
			logger.Info(taskTag, " [detect] 当前在纯香草小岛页")
			return "returnFromIsland", nil
		case IsTaskPage():
			logger.Info(taskTag, " [detect] 当前在任务页")
			return "claimTask", nil
		}
		logger.Info(taskTag, " [detect] 不在已知页面，尝试导航")
		return "navigate", nil
	},

	"navigate": func(sm *core.StateMachine) (string, error) {
		updateHud("导航到活动…")
		if EnsureStarlightHome() {
			return "openManual", nil
		}
		logger.Warn(taskTag, " [navigate] 导航到梦幻繁星岛首页失败")
		return "", errors.New("导航到梦幻繁星岛首页失败")
	},

	"openManual": func(sm *core.StateMachine) (string, error) {
		updateHud("打开航海手册…")
		if !TapSailingManual(1000) {
			return core.RETRY, nil
		}
		if WaitManualPage(10000, 500) {
			return "enterIsland", nil
		}
		logger.Warn(taskTag, " [openManual] 等待航海手册页超时")
		return core.RETRY, nil
	},

	"enterIsland": func(sm *core.StateMachine) (string, error) {
		updateHud("进入纯香草小岛…")
		if !TapLoginIsland(1000) {
			return core.RETRY, nil
		}
		if WaitVanillaIslandPage(10000, 500) {
			return "returnFromIsland", nil
		}
		logger.Warn(taskTag, " [enterIsland] 等待纯香草小岛页超时")
		return core.RETRY, nil
	},

	"returnFromIsland": func(sm *core.StateMachine) (string, error) {
		updateHud("返回首页…")
		if !TapBackFromVanilla(1200) {
			return core.RETRY, nil
		}
		if WaitHomePage(10000, 500) {
			return "openTask", nil
		}
		logger.Warn(taskTag, " [returnFromIsland] 等待首页超时")
		return core.RETRY, nil
	},

	"openTask": func(sm *core.StateMachine) (string, error) {
		updateHud("打开任务页…")
		if !TapTaskBtn(1000) {
			return core.RETRY, nil
		}
		if WaitTaskPage(10000, 500) {
			return "claimTask", nil
		}
		logger.Warn(taskTag, " [openTask] 等待任务页超时")
		return core.RETRY, nil
	},

	"claimTask": func(sm *core.StateMachine) (string, error) {
		updateHud("领取任务奖励…")
		x, y, ok := FindClaimableBtn()
		if ok {
			logger.Info(taskTag, " [claimTask] 发现可领奖按钮 (%d,%d)", x, y)
			TapClaimableBtn(x, y, 800)
			color.Sleep(2000, 500)
			DismissRewardPopupIfNeeded()
		} else {
			logger.Info(taskTag, " [claimTask] 无可领奖按钮")
		}

		MarkDoneToday()
		return "finish", nil
	},

	"finish": func(sm *core.StateMachine) (string, error) {
		updateHud("返回首页…")
		if !TapBackFromTask(1200) {
			return core.RETRY, nil
		}
		if !WaitHomePage(10000, 500) {
			logger.Warn(taskTag, " [finish] 等待首页超时")
			return core.RETRY, nil
		}

		updateHud("返回王国…")
		if !TapBackToKingdom(1200) {
			return core.RETRY, nil
		}
		if kingdom.Wait(10000) {
			logger.Info(taskTag, " 任务完成")
			return core.DONE, nil
		}
		logger.Warn(taskTag, " [finish] 等待王国首页超时")
		return core.RETRY, nil
	},
}

// Run 运行梦幻繁星岛任务（对应 Task.run）。
// 返回 nil 表示完成，其他错误为任务失败。
func Run(g *core.Guard) error {
	logger.Info(taskTag, " 任务启动")
	updateHud("启动…")

	sm := core.New()
	sm.Init("check", core.InitOpts{MaxRetry: 3, TimeoutSec: 180, RetryIntervalMs: 1000})

	ok, err := sm.Run(starlightHandlers, core.RunOpts{Interval: 500, Guard: func() { g.Check() }, Label: "梦幻繁星岛"})
	if ok {
		logger.Info(taskTag, " 任务完成")
		updateHud("今日已完成")
		return nil
	}
	logger.Warn(taskTag, " 任务结束：%v", err)
	updateHud("失败: " + err.Error())
	return err
}
