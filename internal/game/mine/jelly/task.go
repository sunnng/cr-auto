// Package jelly 对应 Lua 工程的 game/常规_未知的地底矿山/模块_解除洋菜冻/：解除洋菜冻页面与会话。
package jelly

import (
	"errors"

	"app/internal/config"
	"app/internal/core"
	"app/internal/game/kingdom"
	"app/internal/game/mine"
	"app/internal/game/mine/route"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/userconfig"
)

const taskTag = "[解除洋菜冻]"

func mineConfig() config.MineConfig {
	cfg, err := userconfig.Mine()
	if err != nil {
		logger.Warn(taskTag, "读取矿山配置失败: %v", err)
		return config.Static.User.Mine
	}
	return cfg
}

// jellyCtx 状态机上下文（对应 Lua 任务 ctx；JellyRemainSec<=0 表示未识别到剩余时间）。
type jellyCtx struct {
	JellyRemainSec int
}

func detect(sm *core.StateMachine) (string, error) {
	switch {
	case IsJellyPage():
		logger.Info(taskTag, " [detect] 当前在解除洋菜冻页面")
		return "processPage", nil
	case mine.IsCurrent():
		logger.Info(taskTag, " [detect] 当前在矿山首页")
		return "enterJelly", nil
	case kingdom.IsKingdomHome():
		logger.Info(taskTag, " [detect] 当前在王国首页")
		return "navigate", nil
	}
	logger.Warn(taskTag, " [detect] 页面识别失败")
	return "", errors.New("解除洋菜冻[detect] 页面识别失败")
}

func navigate(sm *core.StateMachine) (string, error) {
	logger.Info(taskTag, " [navigate] 王国首页 → 矿山首页")
	if route.KingdomHomeToMineHome() {
		return "enterJelly", nil
	}
	logger.Warn(taskTag, " [navigate] 导航到矿山首页失败")
	return "", errors.New("解除洋菜冻[navigate] 导航失败")
}

func enterJelly(sm *core.StateMachine) (string, error) {
	logger.Info(taskTag, " [enterJelly] 矿山首页 → 解除洋菜冻页面")
	TapEnterBtn()
	if WaitJellyPage(30000, 500) {
		return "processPage", nil
	}
	logger.Warn(taskTag, " [enterJelly] 等待解除洋菜冻页面超时")
	return core.RETRY, nil
}

func processPage(sm *core.StateMachine) (string, error) {
	logger.Info(taskTag, " [processPage] 处理解除洋菜冻页面")

	// 1. 可全部领取则先领取并结算。
	if CanClaimAll() {
		logger.Info(taskTag, " [processPage] 检测到可全部领取")
		TapClaimAll()
		color.Sleep(1000, 500)
		ok, _ := color.TapUntilMatch(features.SettleBtn, features.Feature,
			color.TapOpts{TimeoutMs: 30000, IntervalMs: 500, TapDelayMs: 800, SleepMs: 800})
		if !ok {
			logger.Warn(taskTag, " [processPage] 点击 settleBtn 后页面未恢复")
			return core.RETRY, nil
		}
	}

	// 2. OCR 查找「配置」按钮。
	if x, y, found := FindConfigBtn(); found {
		logger.Info(taskTag, " [processPage] 找到配置按钮，进入配置界面")
		TapConfigBtn(x, y)
		if !WaitConfigPage(2000, 300) {
			logger.Info(taskTag, " [processPage] 点击配置后未进入配置洋菜冻页面，无可选择洋菜冻，结束任务")
			sm.Ctx.(*jellyCtx).JellyRemainSec = 0
			return "returnHome", nil
		}
		return "configJelly", nil
	}

	// 3. 无配置按钮：识别剩余时间。
	logger.Info(taskTag, " [processPage] 未找到配置按钮，准备识别剩余时间")
	remainSec, _ := ReadRemainTime()
	sm.Ctx.(*jellyCtx).JellyRemainSec = remainSec
	return "returnHome", nil
}

func configJelly(sm *core.StateMachine) (string, error) {
	logger.Info(taskTag, " [configJelly] 处理配置洋菜冻界面")

	if CanChoose() {
		logger.Info(taskTag, " [configJelly] 可选择，点击选择按钮")
		TapChoose()
		color.Sleep(1000, 500)
		if WaitJellyPage(30000, 500) {
			return "processPage", nil
		}
		logger.Warn(taskTag, " [configJelly] 选择后等待解除洋菜冻页面超时")
		return core.RETRY, nil
	}

	// 不可选择：返回解除洋菜冻页面，再走返回链结束。
	logger.Info(taskTag, " [configJelly] 不可选择，返回解除洋菜冻页面")
	sm.Ctx.(*jellyCtx).JellyRemainSec = 0
	TapConfigBack()
	if !WaitJellyPage(30000, 500) {
		logger.Warn(taskTag, " [configJelly] 返回后等待解除洋菜冻页面超时")
		return core.RETRY, nil
	}
	return "returnHome", nil
}

func returnHome(sm *core.StateMachine) (string, error) {
	logger.Info(taskTag, " [returnHome] 返回王国首页")

	// 统一记录冷却，避免任务立即被再次调度。
	remainSec := sm.Ctx.(*jellyCtx).JellyRemainSec
	if remainSec > 0 {
		EnterWait(remainSec)
	} else {
		EnterWait(0)
	}

	// 解除洋菜冻页 → 矿山首页。
	if IsJellyPage() {
		TapBack()
		if !mine.Wait() {
			logger.Warn(taskTag, " [returnHome] 返回矿山首页超时")
			return core.RETRY, nil
		}
	}

	// 矿山首页 → 王国首页。
	if mine.IsCurrent() {
		mine.TapBack()
		if !kingdom.Wait(30000) {
			logger.Warn(taskTag, " [returnHome] 返回王国首页超时")
			return core.RETRY, nil
		}
	}

	if kingdom.IsKingdomHome() {
		logger.Info(taskTag, " [returnHome] 已回到王国首页")
		return core.DONE, nil
	}

	logger.Warn(taskTag, " [returnHome] 未知页面，无法返回")
	return core.RETRY, nil
}

var jellyHandlers = map[string]core.StateHandler{
	"detect":      detect,
	"navigate":    navigate,
	"enterJelly":  enterJelly,
	"processPage": processPage,
	"configJelly": configJelly,
	"returnHome":  returnHome,
}

// Run 运行解除洋菜冻任务（对应 JellyTask.run）。
// 返回 nil 表示完成，core.ErrSkip 表示任务未启用，其他错误为任务失败。
func Run(g *core.Guard) error {
	cfg := mineConfig()
	if !cfg.JellyEnabled {
		logger.Info(taskTag, " 任务未启用，跳过")
		return core.ErrSkip
	}

	logger.Info(taskTag, "任务启动 | jellyEnabled=%v jellyIntervalSec=%d", cfg.JellyEnabled, cfg.JellyIntervalSec)

	sm := core.New()
	sm.Init("detect", core.InitOpts{MaxRetry: 3, TimeoutSec: 1800, RetryIntervalMs: 1000})
	sm.Ctx = &jellyCtx{}

	ok, err := sm.Run(jellyHandlers, core.RunOpts{Interval: 500, Guard: func() { g.Check() }, Label: "解除洋菜冻"})
	if ok {
		logger.Info(taskTag, " 任务完成")
		return nil
	}
	logger.Warn(taskTag, "任务结束：%v", err)
	return err
}
