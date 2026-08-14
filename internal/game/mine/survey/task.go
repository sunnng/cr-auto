// Package survey 对应 Lua 工程的 game/常规_未知的地底矿山/模块_矿山勘查/：勘查页面与会话。
package survey

import (
	"errors"
	"fmt"
	"time"

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

const taskTag = "[矿山勘查]"

func nowUnix() int64 { return time.Now().Unix() }

// surveyCtx 状态机上下文（对应 Lua 任务 ctx）。
type surveyCtx struct {
	MineVentureCfg config.MineConfig
	NextOcrPollAt  int64
	LastFloor      int
	LastGap        int
}

func mineConfig() config.MineConfig {
	cfg, err := userconfig.Mine()
	if err != nil {
		logger.Warn(taskTag, "读取矿山配置失败: %v", err)
		return config.Static.User.Mine
	}
	return cfg
}

// updateHud 刷新顶部 HUD（对应 Lua updateHud）。
func updateHud(sm *core.StateMachine, patch func(*status.MineSurvey)) {
	ctx := sm.Ctx.(*surveyCtx)
	opts := status.MineSurvey{
		State:      sm.GetState(),
		Floor:      ctx.LastFloor,
		Gap:        ctx.LastGap,
		Target:     ctx.MineVentureCfg.TargetFloor,
		FarGap:     ctx.MineVentureCfg.FarGap,
		Retry:      sm.Retries(),
		FarWaitSec: RestoreProgress(),
	}
	if ctx.NextOcrPollAt > 0 {
		opts.OcrInSec = int(ctx.NextOcrPollAt - nowUnix())
	}
	if patch != nil {
		patch(&opts)
	}
	status.SetMineSurvey(opts)
}

// resolveInitialState 根据持久化进度和当前页面决定起点（对应 Lua resolveInitialState）。
// 返回 (initialState, remainSec, shouldRun)；远距等待未到期时 shouldRun=false。
func resolveInitialState(cfg config.MineConfig) (string, int, bool) {
	if remain := RestoreProgress(); remain > 0 {
		return "", remain, false
	}
	if IsMineVentureDomain() {
		logger.Info(taskTag, " 初始状态: prepare（已在勘查域）")
		return "prepare", 0, true
	}
	if mine.IsCurrent() || kingdom.IsKingdomHome() {
		logger.Info(taskTag, " 初始状态: navigate（在矿山首页或王国首页）")
		return "navigate", 0, true
	}
	logger.Info(taskTag, " 初始状态: detect（页面未知）")
	return "detect", 0, true
}

var surveyHandlers = map[string]core.StateHandler{
	"detect": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineSurvey) { o.Extra = "识别页面…" })
		logger.Debug(taskTag, " [detect] 识别当前页面")
		if IsMineVentureDomain() {
			logger.Info(taskTag, " [detect] 在勘查域 → prepare")
			return "prepare", nil
		}
		if kingdom.IsKingdomHome() || mine.IsCurrent() {
			logger.Info(taskTag, " [detect] 在王国/矿山首页 → navigate")
			return "navigate", nil
		}
		logger.Warn(taskTag, " [detect] 页面识别失败")
		return "", errors.New("矿山勘查[detect] 页面识别失败")
	},

	"navigate": func(sm *core.StateMachine) (string, error) {
		step := "定位中"
		switch {
		case kingdom.IsKingdomHome():
			step = "王国→矿山"
		case mine.IsCurrent():
			step = "矿山→勘查"
		case IsMineVentureDomain():
			step = "已到勘查域"
		}
		updateHud(sm, func(o *status.MineSurvey) { o.Extra = step })
		logger.Debug(taskTag, " [navigate] 导航中")
		switch {
		case kingdom.IsKingdomHome():
			logger.Info(taskTag, " [navigate] 王国首页 → 矿山首页")
			route.KingdomHomeToMineHome()
		case mine.IsCurrent():
			logger.Info(taskTag, " [navigate] 矿山首页 → 进入勘查")
			mine.TapVenture()
		case IsMineVentureDomain():
			logger.Info(taskTag, " [navigate] 已进入勘查域 → prepare")
			color.Sleep(1000, 500)
			return "prepare", nil
		}
		return core.KEEP, nil
	},

	"prepare": func(sm *core.StateMachine) (string, error) {
		if IsRunning() {
			updateHud(sm, func(o *status.MineSurvey) { o.Extra = "勘查进行中" })
		} else {
			updateHud(sm, func(o *status.MineSurvey) { o.Extra = "启动 setup…" })
		}
		logger.Debug(taskTag, " [prepare] 准备勘查")
		if IsRunning() {
			logger.Info(taskTag, " [prepare] 勘查进行中 → running")
			return "running", nil
		}
		if Setup() {
			waitSec := sm.Ctx.(*surveyCtx).MineVentureCfg.FarWaitSec
			EnterFarWait(waitSec)
			updateHud(sm, func(o *status.MineSurvey) {
				o.FarWaitSec = waitSec
				o.Extra = "已启动 回城"
			})
			logger.Info(taskTag, " [prepare] setup 完成 → farWait")
			return "farWait", nil
		}
		logger.Warn(taskTag, " [prepare] setup 执行失败")
		return "", errors.New("矿山勘查[prepare] setup 执行失败")
	},

	"running": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineSurvey) { o.Extra = "OCR 读层…" })
		logger.Debug(taskTag, " [running] 读取当前层数")
		currentFloor, ok := GetCurrentFloor()
		if !ok {
			updateHud(sm, func(o *status.MineSurvey) { o.Extra = "OCR 失败" })
			logger.Warn(taskTag, " [running] OCR 未识别层数，重试")
			return core.RETRY, nil
		}

		cfg := sm.Ctx.(*surveyCtx).MineVentureCfg
		targetFloor := cfg.TargetFloor
		floorDiff := abs(targetFloor - currentFloor)
		ctx := sm.Ctx.(*surveyCtx)
		ctx.LastFloor = currentFloor
		ctx.LastGap = floorDiff

		logger.Info(taskTag, "[running] 当前层:%d 目标:%d 阈值:%d 轮询:%ds 远距等待:%ds",
			currentFloor, targetFloor, cfg.FarGap, cfg.OcrPollSec, cfg.FarWaitSec)

		switch {
		case currentFloor >= targetFloor:
			updateHud(sm, func(o *status.MineSurvey) { o.Extra = "已达标" })
			logger.Info(taskTag, " [running] 已达标 → settle")
			return "settle", nil
		case floorDiff > cfg.FarGap:
			EnterFarWait(cfg.FarWaitSec)
			updateHud(sm, func(o *status.MineSurvey) {
				o.FarWaitSec = cfg.FarWaitSec
				o.Extra = fmt.Sprintf("远距 差%d>%d", floorDiff, cfg.FarGap)
			})
			logger.Info(taskTag, "[running] 远距(差%d>%d) → farWait，回城等待 %ds",
				floorDiff, cfg.FarGap, cfg.FarWaitSec)
			return "farWait", nil
		default:
			updateHud(sm, func(o *status.MineSurvey) {
				o.Extra = fmt.Sprintf("近距 差%d≤%d", floorDiff, cfg.FarGap)
			})
			logger.Info(taskTag, "[running] 近距(差%d<=%d) → polling", floorDiff, cfg.FarGap)
			return "polling", nil
		}
	},

	"polling": func(sm *core.StateMachine) (string, error) {
		cfg := sm.Ctx.(*surveyCtx).MineVentureCfg
		ctx := sm.Ctx.(*surveyCtx)

		if ctx.NextOcrPollAt == 0 {
			ctx.NextOcrPollAt = nowUnix() + int64(cfg.OcrPollSec)
			logger.Debug(taskTag, "[polling] 首次进入，下次 OCR 在 %ds 后", cfg.OcrPollSec)
		}

		if nowUnix() >= ctx.NextOcrPollAt {
			updateHud(sm, func(o *status.MineSurvey) { o.Extra = "OCR 轮询…" })
			currentFloor, ok := GetCurrentFloor()
			ctx.NextOcrPollAt = nowUnix() + int64(cfg.OcrPollSec)
			if ok {
				ctx.LastFloor = currentFloor
				ctx.LastGap = abs(cfg.TargetFloor - currentFloor)
			}
			switch {
			case ok && currentFloor >= cfg.TargetFloor:
				updateHud(sm, func(o *status.MineSurvey) { o.Extra = "轮询达标" })
				logger.Info(taskTag, "[polling] 达标 当前层:%d ≥ 目标:%d → settle", currentFloor, cfg.TargetFloor)
				return "settle", nil
			case !ok:
				updateHud(sm, func(o *status.MineSurvey) { o.Extra = "OCR 失败" })
				logger.Warn(taskTag, " [polling] OCR 未识别层数，重试")
				return core.RETRY, nil
			default:
				updateHud(sm, func(o *status.MineSurvey) {
					o.Floor = ctx.LastFloor
					o.Gap = ctx.LastGap
					o.OcrInSec = cfg.OcrPollSec
					o.Extra = "未达标"
				})
				logger.Debug(taskTag, "[polling] 当前层:%d 目标:%d，继续等待", currentFloor, cfg.TargetFloor)
				return core.KEEP, nil
			}
		}

		updateHud(sm, func(o *status.MineSurvey) {
			o.Extra = "等待 OCR"
		})
		logger.Debug(taskTag, " [polling] 等待下次 OCR 轮询点")
		return core.KEEP, nil
	},

	"settle": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineSurvey) { o.Extra = "停止并结算…" })
		logger.Info(taskTag, " [settle] 停止勘查并结算")
		if StopVenture() {
			updateHud(sm, func(o *status.MineSurvey) { o.Extra = "结算完成" })
			logger.Info(taskTag, " [settle] 结算完成 → detect（进入下一轮识别）")
			return "detect", nil
		}
		updateHud(sm, func(o *status.MineSurvey) { o.Extra = "结算失败" })
		logger.Warn(taskTag, " [settle] 停止勘查失败")
		return core.RETRY, nil
	},

	"farWait": func(sm *core.StateMachine) (string, error) {
		updateHud(sm, func(o *status.MineSurvey) { o.Extra = "回城中…" })
		logger.Info(taskTag, " [farWait] 导航回王国，本轮结束（等待期满后由调度再次拉起）")
		BackToMineHome()
		route.MineHomeToKingdom()
		remain := RestoreProgress()
		updateHud(sm, func(o *status.MineSurvey) {
			o.State = "idle"
			if remain > 0 {
				o.FarWaitSec = remain
			} else {
				o.FarWaitSec = sm.Ctx.(*surveyCtx).MineVentureCfg.FarWaitSec
			}
			o.Extra = "本轮结束"
		})
		return core.DONE, nil
	},
}

// Run 运行矿山勘查任务（对应 MineVentureTask.run）。
// 返回 nil 表示完成，core.ErrSkip 表示本轮跳过（未启用/远距等待中），其他错误为任务失败。
func Run(g *core.Guard) error {
	cfg := mineConfig()
	logger.Info(taskTag, "任务启动 | 目标层:%d 近距阈值:%d 轮询:%ds 远距等待:%ds",
		cfg.TargetFloor, cfg.FarGap, cfg.OcrPollSec, cfg.FarWaitSec)

	initialState, remain, shouldRun := resolveInitialState(cfg)
	if !shouldRun {
		status.SetMineSurvey(status.MineSurvey{
			State:      "idle",
			Target:     cfg.TargetFloor,
			FarGap:     cfg.FarGap,
			FarWaitSec: remain,
			Extra:      "远距等待中",
		})
		logger.Info(taskTag, "远距等待中，剩余 %ds，本轮跳过", remain)
		return core.ErrSkip
	}

	status.SetMineSurvey(status.MineSurvey{
		State:   initialState,
		Target:  cfg.TargetFloor,
		FarGap:  cfg.FarGap,
		CfgHint: fmt.Sprintf("轮询%ds 远距%ds", cfg.OcrPollSec, cfg.FarWaitSec),
		Extra:   "任务启动",
	})

	sm := core.New()
	sm.Init(initialState, core.InitOpts{MaxRetry: 3, TimeoutSec: 1800, RetryIntervalMs: 1000})
	sm.Ctx = &surveyCtx{MineVentureCfg: cfg}

	ok, err := sm.Run(surveyHandlers, core.RunOpts{Interval: 500, Guard: func() { g.Check() }, Label: "矿山勘查"})
	if ok {
		logger.Info(taskTag, " 任务完成")
		return nil
	}
	status.SetMineSurvey(status.MineSurvey{
		Target:     cfg.TargetFloor,
		FarGap:     cfg.FarGap,
		FarWaitSec: RestoreProgress(),
		Extra:      "失败: " + err.Error(),
	})
	logger.Warn(taskTag, "任务结束：%v", err)
	return err
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
