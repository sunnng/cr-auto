// Package mining 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山开采/：开采页面与会话。
package mining

import (
	"errors"
	"fmt"

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

const taskTag = "[矿山开采]"

var defaultCardPriority = []string{
	"butterAmber", "amberFossil", "sugarOre", "purpleFossil", "emeraldFossil", "flourStone",
}

func mineConfig() config.MineConfig {
	cfg, err := userconfig.Mine()
	if err != nil {
		logger.Warn(taskTag, "读取矿山配置失败: %v", err)
		return config.Static.User.Mine
	}
	return cfg
}

// resolveCardPriority 解析矿卡选卡优先级（对应 Lua resolveCardPriority）。
func resolveCardPriority() []string {
	cfg := mineConfig()
	keys := cfg.MiningOreCards
	if len(keys) == 0 {
		keys = defaultCardPriority
	}
	var out []string
	for _, key := range keys {
		if hasFindDef(mine.OreVeinCards[key]) {
			out = append(out, key)
		}
	}
	if len(out) == 0 {
		for _, key := range defaultCardPriority {
			if hasFindDef(mine.OreVeinCards[key]) {
				out = append(out, key)
			}
		}
	}
	return out
}

// miningCtx 状态机上下文（对应 Lua 任务 ctx）。
type miningCtx struct {
	QuotaCur           int
	QuotaMax           int
	SelectedCards      int
	CardSwipeDirection string
	SkipSelectOnce     bool
}

// updateHud 刷新顶部 HUD（对应 Lua updateHud）。
func updateHud(sm *core.StateMachine, patch func(*status.MineMining)) {
	ctx := sm.Ctx.(*miningCtx)
	opts := status.MineMining{
		State:    sm.GetState(),
		Selected: ctx.SelectedCards,
		Quota:    ctx.QuotaMax,
		Retry:    sm.Retries(),
	}
	if patch != nil {
		patch(&opts)
	}
	status.SetMineMining(opts)
}

var miningHandlers = map[string]core.StateHandler{
	"detect": func(sm *core.StateMachine) (string, error) {
		switch {
		case IsMiningPage():
			return "miningPageScan", nil
		case mine.IsCurrent():
			return "precheck", nil
		case kingdom.IsKingdomHome():
			return "navigate", nil
		}
		logger.Warn(taskTag, " [detect] 页面识别失败")
		return "", errors.New("矿山开采[detect] 页面识别失败")
	},

	"navigate": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "王国→矿山首页" })
		if route.KingdomHomeToMineHome() {
			return "precheck", nil
		}
		logger.Warn(taskTag, " [navigate] 王国→矿山首页失败")
		return "", errors.New("矿山开采[navigate] 王国→矿山首页失败")
	},

	"precheck": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "首页预检…" })

		if mine.HasMiningCompletedTask() {
			logger.Info(taskTag, " [precheck] 首页发现开采存在已完成开采任务 → 进入开采页")
		}

		mine.TapMining()
		if mine.WaitGone() {
			color.Sleep(1000, 500)
			if IsMiningPage() {
				return "miningPageScan", nil
			}
			if IsSettlementRoute() {
				return "confirmRewards", nil
			}
		}

		logger.Warn(taskTag, " [precheck] 未能进入开采页面")
		return core.RETRY, nil
	},

	"miningPageScan": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "扫描开采页面…" })

		if TapCompletedSlot() {
			return "confirmRewards", nil
		}

		ctx := sm.Ctx.(*miningCtx)
		if ctx.SkipSelectOnce {
			ctx.SkipSelectOnce = false
			logger.Info(taskTag, " [miningPageScan] 无可用矿卡可填栏位，跳过选卡")
		} else if TapFreeSlot() {
			logger.Info(taskTag, " [miningPageScan] 有空闲栏位 → selectMineCard")
			return "selectMineCard", nil
		} else if WasNoMineCard() {
			logger.Info(taskTag, " [miningPageScan] 矿脉卡清单无矿卡，准备回城结束")
			return "noCardReturn", nil
		}

		if TapReadySlot() {
			logger.Info(taskTag, " [miningPageScan] 有可开始矿卡 → startMining")
			return "startMining", nil
		}

		logger.Info(taskTag, " [miningPageScan] 无已完成/无空闲/无可开采栏位 → done")
		return "done", nil
	},

	"confirmRewards": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "获得开采奖励, 点击画面继续…" })

		if TapUntilMatchMiningPage() {
			color.Sleep(1000, 500)
			logger.Info(taskTag, " [claimConfirm] 奖励已确认 → miningPageScan")
			return "miningPageScan", nil
		}

		logger.Warn(taskTag, " [claimConfirm] 确认奖励失败")
		return core.RETRY, nil
	},

	"startMining": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "开始开采矿卡" })

		if AutoSelectCookieAndStart() {
			color.Sleep(1500, 500)
			if IsSetup() {
				return core.RETRY, nil
			}
			return "miningPageScan", nil
		}

		logger.Warn(taskTag, " [startMining] 开采矿卡失败")
		return core.RETRY, nil
	},

	"selectMineCard": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "选择矿卡" })

		initCur, initMax, initRaw, ok := ReadChooseQuota()
		if !ok {
			logger.Warn(taskTag, "[selectMineCard] OCR 可选数量失败 raw=%s", initRaw)
			return core.RETRY, nil
		}

		ctx := sm.Ctx.(*miningCtx)
		ctx.QuotaCur = initCur
		ctx.QuotaMax = initMax
		ctx.SelectedCards = initCur
		updateHud(sm, func(o *status.MineMining) {
			o.Selected = initCur
			o.Quota = initMax
			o.Extra = fmt.Sprintf("选卡 %d/%d", initCur, initMax)
		})

		if initCur >= initMax {
			logger.Info(taskTag, "[selectMineCard] 初始已选满 %d/%d，直接确认", initCur, initMax)
		} else {
			cardPriority := resolveCardPriority()
			direction := ctx.CardSwipeDirection
			if direction == "" {
				direction = "left"
			}
			for _, cardKey := range cardPriority {
				cur, max, _, ok := ReadChooseQuota()
				if !ok {
					logger.Warn(taskTag, " [selectMineCard] 切换目标前 OCR 失败")
					return core.RETRY, nil
				}
				if cur >= max {
					break
				}
				need := max - cur
				cardDef := mine.OreVeinCards[cardKey]
				if hasFindDef(cardDef) {
					logger.Info(taskTag, "[selectMineCard] 目标矿卡 %s 方向:%s 还需 %d 张", cardKey, direction, need)

					got, exhausted := SelectTargetCards(cardDef, need, direction)
					cur, max, _, ok = ReadChooseQuota()
					if !ok {
						logger.Warn(taskTag, " [selectMineCard] 选卡后 OCR 失败")
						return core.RETRY, nil
					}

					ctx.QuotaCur = cur
					ctx.QuotaMax = max
					ctx.SelectedCards = cur
					updateHud(sm, func(o *status.MineMining) {
						o.Selected = cur
						o.Quota = max
						o.Extra = fmt.Sprintf("选卡 %d/%d (%s+%d)", cur, max, cardKey, got)
					})

					if cur >= max {
						logger.Info(taskTag, "[selectMineCard] 已选满 %d/%d", cur, max)
						break
					}

					if exhausted || got == 0 {
						if exhausted {
							if direction == "left" {
								direction = "right"
							} else {
								direction = "left"
							}
							ctx.CardSwipeDirection = direction
						}
						logger.Info(taskTag, "[selectMineCard] %s 已扫完/无新增（+%d），切换下一种，方向:%s", cardKey, got, direction)
					} else {
						logger.Warn(taskTag, " [selectMineCard] 有新增但未填满，重试当前选卡流程")
						return core.RETRY, nil
					}
				} else {
					logger.Debug(taskTag, " [selectMineCard] 跳过未配置矿卡 %s", cardKey)
				}
			}
		}

		finalCur, finalMax, finalRaw, ok := ReadChooseQuota()
		if !ok {
			logger.Warn(taskTag, "[selectMineCard] 最终 OCR 校验失败 raw=%s", finalRaw)
			return core.RETRY, nil
		}

		ctx.QuotaCur = finalCur
		ctx.QuotaMax = finalMax
		ctx.SelectedCards = finalCur
		updateHud(sm, func(o *status.MineMining) {
			o.Selected = finalCur
			o.Quota = finalMax
			o.Extra = fmt.Sprintf("选卡完成 %d/%d", finalCur, finalMax)
		})

		if finalCur <= 0 {
			logger.Warn(taskTag, " [selectMineCard] 未选择任何矿卡，返回开采页")
			TapBackBtn()
			color.Sleep(800, 500)
			ctx.SkipSelectOnce = true
			return "miningPageScan", nil
		}

		if finalCur < finalMax {
			logger.Info(taskTag, "[selectMineCard] 配额未填满 %d/%d，确认已有选择", finalCur, finalMax)
		}

		updateHud(sm, func(o *status.MineMining) { o.Extra = "确认选卡…" })
		if !ConfirmCardSelection() {
			logger.Warn(taskTag, " [selectMineCard] 确认选卡失败")
			if IsMiningPage() {
				return "miningPageScan", nil
			}
			return core.RETRY, nil
		}

		color.Sleep(1000, 500)
		if !WaitMiningPage(30000, 500) {
			logger.Warn(taskTag, " [selectMineCard] 等待开采页超时")
			return core.RETRY, nil
		}

		logger.Info(taskTag, " [selectMineCard] 已确认 → startMining")
		return "startMining", nil
	},

	"noCardReturn": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "无矿卡可选，回城结束…" })
		logger.Info(taskTag, " [noCardReturn] 矿脉卡清单无矿卡，准备回城")
		if !ReturnToKingdom() {
			logger.Warn(taskTag, " [noCardReturn] 回王国首页失败")
			return "", errors.New("矿山开采[noCardReturn] 回王国首页失败")
		}
		EnterBusyWait(0)
		return core.DONE, nil
	},

	"done": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineMining) { o.Extra = "本轮结束回城…" })
		// 结束前复查一次，避免识别抖动导致误判全忙。
		if HasCompletedTask() {
			logger.Info(taskTag, " [done] 复查发现已完成任务 → confirmRewards")
			TapCompletedSlot()
			return "confirmRewards", nil
		}
		if HasFreeSlot() {
			logger.Info(taskTag, " [done] 复查发现空闲栏位 → selectMineCard")
			TapFreeSlot()
			return "selectMineCard", nil
		}
		if HasStartableCard() {
			logger.Info(taskTag, " [done] 复查发现可开采矿卡 → startMining")
			TapReadySlot()
			return "startMining", nil
		}

		logger.Info(taskTag, " [done] 当前页无可操作项，准备回城并记录 busy")
		updateHud(sm, func(o *status.MineMining) {
			o.State = "idle"
			o.Extra = "本轮结束"
		})
		if !ReturnToKingdom() {
			logger.Warn(taskTag, " [done] 回王国首页失败")
			return "", errors.New("矿山开采[done] 回王国首页失败")
		}
		EnterBusyWait(0)
		return core.DONE, nil
	},
}

// Run 运行矿山开采任务（对应 MiningTask.run）。
// 返回 nil 表示完成，core.ErrSkip 表示任务未启用，其他错误为任务失败。
func Run(g *core.Guard) error {
	cfg := mineConfig()
	if !cfg.MiningEnabled {
		logger.Info(taskTag, " 任务未启用，跳过")
		return core.ErrSkip
	}
	cardPriority := resolveCardPriority()
	logger.Info(taskTag, "任务启动 | miningEnabled=%v oreCards=%v", cfg.MiningEnabled, cardPriority)

	status.SetMineMining(status.MineMining{
		State: "detect",
		Extra: "任务启动",
	})

	sm := core.New()
	sm.Init("detect", core.InitOpts{MaxRetry: 3, TimeoutSec: 1800, RetryIntervalMs: 1000})
	sm.Ctx = &miningCtx{CardSwipeDirection: "left"}

	ok, err := sm.Run(miningHandlers, core.RunOpts{Interval: 500, Guard: func() { g.Check() }, Label: "矿山开采"})
	if ok {
		logger.Info(taskTag, " 任务完成")
		return nil
	}
	status.SetMineMining(status.MineMining{Extra: "失败: " + err.Error()})
	logger.Warn(taskTag, "任务结束：%v", err)
	return err
}

// ReturnToKingdom 矿山相关任意页面 → 王国首页（对应 Route.returnToKingdom；
// 因 Go 包循环限制随唯一使用方矿山开采存放）。
func ReturnToKingdom() bool {
	if kingdom.IsKingdomHome() {
		return true
	}

	if IsMiningPage() || IsRewardPage() || IsSettlementRoute() {
		TapBackBtn()
		if !mine.Wait() {
			logger.Warn(taskTag, " 开采页返回矿山首页超时")
		}
	}

	if mine.IsCurrent() {
		if route.MineHomeToKingdom() {
			logger.Info(taskTag, " 已回王国首页")
			return true
		}
		logger.Warn(taskTag, " 矿山首页返回王国超时")
		return false
	}

	if kingdom.IsKingdomHome() {
		return true
	}

	logger.Warn(taskTag, " 回王国首页失败，当前页面未知")
	return false
}
