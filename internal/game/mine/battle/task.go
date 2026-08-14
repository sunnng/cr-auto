// Package battle 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山战斗/：战斗页面与会话。
package battle

import (
	"errors"
	"fmt"
	"image"

	"app/internal/config"
	"app/internal/core"
	"app/internal/game/kingdom"
	"app/internal/game/mine"
	"app/internal/game/mine/route"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/status"
	"app/internal/lib/userconfig"
)

const taskTag = "[矿山战斗]"

func mineConfig() config.MineConfig {
	cfg, err := userconfig.Mine()
	if err != nil {
		logger.Warn(taskTag, "读取矿山配置失败: %v", err)
		return config.Static.User.Mine
	}
	return cfg
}

// resolveTargetSoulStones 解析目标灵魂石集合（对应 Lua resolveTargetSoulStones）。
func resolveTargetSoulStones() map[string]bool {
	cfg := mineConfig()
	out := map[string]bool{}
	for _, name := range cfg.BattleSoulStones {
		out[name] = true
	}
	return out
}

// battleCtx 状态机上下文（对应 Lua 任务 ctx）。
type battleCtx struct {
	QuickBattlePoint image.Point
	HasQuickBattle   bool
}

// updateHud 刷新顶部 HUD（对应 Lua updateHud）。
func updateHud(sm *core.StateMachine, patch func(*status.MineBattle)) {
	opts := status.MineBattle{
		State: sm.GetState(),
		Retry: sm.Retries(),
	}
	if patch != nil {
		patch(&opts)
	}
	status.SetMineBattle(opts)
}

func detect(sm *core.StateMachine) (string, error) {
	updateHud(sm, func(o *status.MineBattle) { o.Extra = "识别页面…" })

	switch {
	case IsBattlePage():
		logger.Info(taskTag, " [detect] 在矿山战斗页 → battleLoop")
		return "battleLoop", nil
	case mine.IsCurrent():
		logger.Info(taskTag, " [detect] 在矿山首页 → navigate")
		return "navigate", nil
	case kingdom.IsKingdomHome():
		logger.Info(taskTag, " [detect] 在王国首页 → navigate")
		return "navigate", nil
	}

	logger.Warn(taskTag, " [detect] 页面识别失败")
	return "", errors.New("矿山战斗[detect] 页面识别失败")
}

func navigate(sm *core.StateMachine) (string, error) {
	if kingdom.IsKingdomHome() {
		updateHud(sm, func(o *status.MineBattle) { o.Extra = "王国→矿山首页" })
		logger.Info(taskTag, " [navigate] 王国首页 → 矿山首页")
		if !route.KingdomHomeToMineHome() {
			logger.Warn(taskTag, " [navigate] 王国→矿山首页失败")
			return core.RETRY, nil
		}
		return core.KEEP, nil
	}

	if mine.IsCurrent() {
		updateHud(sm, func(o *status.MineBattle) { o.Extra = "矿山首页→战斗页" })
		logger.Info(taskTag, " [navigate] 矿山首页 → 战斗页")
		mine.TapBattleBtn()
		if WaitBattlePage(30000, 500) {
			return "battleLoop", nil
		}
		logger.Warn(taskTag, " [navigate] 等待矿山战斗页超时")
		return core.RETRY, nil
	}

	if IsBattlePage() {
		return "battleLoop", nil
	}

	logger.Warn(taskTag, " [navigate] 当前页面未知，无法导航")
	return "", errors.New("矿山战斗[navigate] 当前页面未知")
}

// quickBattle 快转流程（对应 Lua quickBattle）。
func quickBattle(sm *core.StateMachine) (string, error) {
	updateHud(sm, func(o *status.MineBattle) { o.Extra = "快转弹窗…" })

	ctx := sm.Ctx.(*battleCtx)
	if !ctx.HasQuickBattle {
		logger.Warn(taskTag, " [quickBattle] 缺少快转按钮坐标")
		return "exit", nil
	}

	TapQuickBattleButton(ctx.QuickBattlePoint.X, ctx.QuickBattlePoint.Y)
	if !WaitQuickBattleDialog(10000, 500) {
		logger.Warn(taskTag, " [quickBattle] 快转弹窗未出现")
		return core.RETRY, nil
	}

	color.Sleep(500, 500)
	used, owned, raw, ok := ReadClockCount()
	logger.Info(taskTag, "[quickBattle] 发条 %s (used=%d owned=%d)", raw, used, owned)

	if !ok {
		logger.Warn(taskTag, " [quickBattle] 发条数量读取失败，取消快转")
		TapQuickBattleCancel()
		WaitQuickBattleDialogGone(5000, 500)
		return "exit", nil
	}

	if used > owned {
		logger.Info(taskTag, " [quickBattle] 发条不足，取消快转")
		TapQuickBattleCancel()
		WaitQuickBattleDialogGone(5000, 500)
		return "exit", nil
	}

	logger.Info(taskTag, " [quickBattle] 发条充足，确认快转")
	TapQuickBattleConfirm()

	if TapSettleUntilBattlePage() {
		logger.Info(taskTag, " [quickBattle] 快转结算完成 → battleLoop")
		return "battleLoop", nil
	}

	logger.Warn(taskTag, " [quickBattle] 结算后未回到战斗页")
	return core.RETRY, nil
}

// scanAndIterateCards 战斗卡扫描与迭代（对应 Lua scanAndIterateCards）。
func scanAndIterateCards(sm *core.StateMachine, targetNames map[string]bool) (string, error) {
	cards := FindBattleCards()
	logger.Info(taskTag, "[battleLoop] 战斗卡数量=%d", len(cards))

	if len(cards) == 1 {
		logger.Info(taskTag, " [battleLoop] 仅1张战斗卡，退出")
		return "exit", nil
	}

	if len(cards) > 1 {
		for i := 1; i < len(cards); i++ {
			updateHud(sm, func(o *status.MineBattle) { o.Extra = fmt.Sprintf("点击战斗卡 %d/%d", i+1, len(cards)) })
			card := cards[i]
			logger.Info(taskTag, "[battleLoop] 点击第 %d/%d 张战斗卡 (%d,%d)", i+1, len(cards), card.X, card.Y)
			TapBattleCard(card)
			color.Sleep(800, 500)

			if matched := RecognizeSoulStoneType(targetNames); matched != "" {
				logger.Info(taskTag, "[battleLoop] 灵魂石匹配: %s", matched)
				if qx, qy, found := FindQuickBattleButton(); found {
					ctx := sm.Ctx.(*battleCtx)
					ctx.QuickBattlePoint = image.Point{X: qx, Y: qy}
					ctx.HasQuickBattle = true
					return "quickBattle", nil
				}
				logger.Warn(taskTag, " [battleLoop] 灵魂石匹配但快转按钮消失，继续迭代")
			} else {
				logger.Debug(taskTag, " [battleLoop] 灵魂石不匹配，继续下一张")
			}
		}
	}

	if len(cards) >= 5 {
		updateHud(sm, func(o *status.MineBattle) { o.Extra = "翻页检查…" })
		logger.Info(taskTag, " [battleLoop] 战斗卡≥5，执行翻页检查")
		if SwipeUpAndCheckLastPage() {
			logger.Info(taskTag, " [battleLoop] 已到末页，退出")
			return "exit", nil
		}
		logger.Info(taskTag, " [battleLoop] 未到末页，重新扫描战斗卡")
		return "battleLoop", nil
	}

	logger.Info(taskTag, " [battleLoop] 战斗卡<5且无可操作项，退出")
	return "exit", nil
}

func battleLoop(sm *core.StateMachine) (string, error) {
	updateHud(sm, func(o *status.MineBattle) { o.Extra = "扫描快转…" })

	if !IsBattlePage() {
		logger.Warn(taskTag, " [battleLoop] 当前不在战斗页")
		return core.RETRY, nil
	}

	targetNames := resolveTargetSoulStones()

	// 1. 优先处理当前页可见的快转。
	if qx, qy, found := FindQuickBattleButton(); found {
		logger.Info(taskTag, " [battleLoop] 发现快转按钮")
		if matched := RecognizeSoulStoneType(targetNames); matched != "" {
			logger.Info(taskTag, "[battleLoop] 快转灵魂石匹配: %s", matched)
			ctx := sm.Ctx.(*battleCtx)
			ctx.QuickBattlePoint = image.Point{X: qx, Y: qy}
			ctx.HasQuickBattle = true
			return "quickBattle", nil
		}
		logger.Info(taskTag, " [battleLoop] 快转灵魂石不匹配，扫描战斗卡")
		return scanAndIterateCards(sm, targetNames)
	}

	// 2. 无快转按钮，扫描战斗卡。
	logger.Info(taskTag, " [battleLoop] 无快转按钮，扫描战斗卡")
	return scanAndIterateCards(sm, targetNames)
}

func exitTask(sm *core.StateMachine) (string, error) {
	updateHud(sm, func(o *status.MineBattle) { o.Extra = "返回矿山首页…" })

	if IsBattlePage() {
		TapBackBtn()
		if !mine.Wait() {
			logger.Warn(taskTag, " [exit] 返回矿山首页超时")
			return core.RETRY, nil
		}
	}

	updateHud(sm, func(o *status.MineBattle) { o.Extra = "返回王国首页…" })
	if mine.IsCurrent() {
		mine.TapBack()
		if !kingdom.Wait(30000) {
			logger.Warn(taskTag, " [exit] 返回王国首页超时")
			return core.RETRY, nil
		}
	}

	if kingdom.IsKingdomHome() {
		logger.Info(taskTag, " [exit] 已回到王国首页")
		return core.DONE, nil
	}

	logger.Warn(taskTag, " [exit] 退出链路失败")
	return core.RETRY, nil
}

var battleHandlers = map[string]core.StateHandler{
	"detect":      detect,
	"navigate":    navigate,
	"battleLoop":  battleLoop,
	"quickBattle": quickBattle,
	"exit":        exitTask,
}

// Run 运行矿山战斗任务（对应 BattleTask.run）。
// 返回 nil 表示完成，core.ErrSkip 表示跳过（未启用/未配置目标），其他错误为任务失败。
func Run(g *core.Guard) error {
	cfg := mineConfig()
	if !cfg.BattleEnabled {
		logger.Info(taskTag, " 任务未启用，跳过")
		return core.ErrSkip
	}

	targetNames := resolveTargetSoulStones()
	if len(targetNames) == 0 {
		logger.Warn(taskTag, " 未配置目标灵魂石，跳过战斗任务")
		return core.ErrSkip
	}

	targetList := make([]string, 0, len(targetNames))
	for name := range targetNames {
		targetList = append(targetList, name)
	}
	logger.Info(taskTag, "任务启动 | battleEnabled=%v targetSoulStones=%v", cfg.BattleEnabled, targetList)

	// 记录本次战斗时间，用于控制检测频率。
	RecordBattle()

	status.SetMineBattle(status.MineBattle{
		State: "detect",
		Extra: "任务启动",
	})

	sm := core.New()
	sm.Init("detect", core.InitOpts{MaxRetry: 3, TimeoutSec: 1800, RetryIntervalMs: 1000})
	sm.Ctx = &battleCtx{}

	ok, err := sm.Run(battleHandlers, core.RunOpts{Interval: 500, Guard: func() { g.Check() }, Label: "矿山战斗"})
	if ok {
		logger.Info(taskTag, " 任务完成")
		return nil
	}
	status.SetMineBattle(status.MineBattle{Extra: "失败: " + err.Error()})
	logger.Warn(taskTag, "任务结束：%v", err)
	return err
}
