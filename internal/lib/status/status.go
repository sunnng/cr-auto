// Package status 对应 Lua 工程的 lib/status-hud.lua 的调度侧接口：主循环/调度器/任务构建器
// 把运行阶段与等待信息发布到宿主（设备端由 main 接到控制面板，桌面测试注入记录器）。
package status

import (
	"fmt"
	"strings"
	"sync"
)

// Phase 运行阶段。
type Phase string

const (
	PhaseRun  Phase = "run"
	PhaseIdle Phase = "idle"
	PhaseWait Phase = "wait"
	PhaseTask Phase = "task"
)

// Update 一次状态发布。
type Update struct {
	Phase Phase
	Task  string // PhaseTask 时的任务名
	Text  string // 等待/任务提示文本
}

// Sink 状态输出端。
type Sink func(Update)

var (
	mu   sync.RWMutex
	sink Sink
)

// SetSink 注入输出端并返回旧输出端；nil 表示丢弃。
func SetSink(fn Sink) Sink {
	mu.Lock()
	defer mu.Unlock()
	prev := sink
	sink = fn
	return prev
}

func publish(update Update) {
	mu.RLock()
	fn := sink
	mu.RUnlock()
	if fn != nil {
		fn(update)
	}
}

// Set 设置主阶段文本（如运行中）。
func Set(phase Phase, text string) { publish(Update{Phase: phase, Text: text}) }

// SetTask 设置当前任务及提示文本。
func SetTask(name, text string) { publish(Update{Phase: PhaseTask, Task: name, Text: text}) }

// SetWait 设置等待提示文本（空闲等待阶段）。
func SetWait(text string) { publish(Update{Phase: PhaseWait, Text: text}) }

// SetIdle 设置空闲挂机状态。
func SetIdle() { publish(Update{Phase: PhaseIdle}) }

// 矿山 HUD 状态标签（对应 Lua status-hud.lua 的 *_STATE_LABEL）。
// 注意：开采标签表与 Lua 原表逐项一致——Lua 侧同样未收录
// miningPageScan/confirmRewards/selectMineCard/startMining/noCardReturn/done，
// 未收录状态按原样展示（结构直译，不擅自扩充）。
var mineSurveyStateLabel = map[string]string{
	"detect": "识别", "navigate": "导航", "prepare": "准备", "running": "读层",
	"polling": "守候", "settle": "结算", "farWait": "远距", "idle": "挂机",
}

var mineMiningStateLabel = map[string]string{
	"detect": "识别", "navigate": "导航", "precheck": "预检", "miningPage": "开采页",
	"claimTap": "领奖", "claimConfirm": "确认", "checkSlot": "检查", "selectFlow": "选卡",
	"startFlow": "启动", "recordDone": "完成", "idle": "等待",
}

var mineBattleStateLabel = map[string]string{
	"detect": "识别", "navigate": "导航", "battleLoop": "扫描", "quickBattle": "快转", "exit": "回城",
}

// MineSurvey 矿山勘查 HUD 字段（0 表示未提供）。
type MineSurvey struct {
	State                                                   string
	Floor, Target, Gap, FarGap, OcrInSec, FarWaitSec, Retry int
	CfgHint                                                 string
	Extra                                                   string
}

// SetMineSurvey 矿山勘查状态发布（对应 Lua StatusHud.setMineSurvey）。
func SetMineSurvey(opts MineSurvey) {
	parts := []string{"矿山勘查"}
	if s := labelOf(mineSurveyStateLabel, opts.State); s != "" {
		parts = append(parts, s)
	}
	if opts.Target > 0 {
		if opts.Floor > 0 {
			if opts.Gap > 0 {
				parts = append(parts, fmt.Sprintf("层%d→%d 差%d", opts.Floor, opts.Target, opts.Gap))
			} else {
				parts = append(parts, fmt.Sprintf("层%d→%d", opts.Floor, opts.Target))
			}
		} else {
			parts = append(parts, fmt.Sprintf("目标%d层", opts.Target))
		}
	}
	if opts.FarGap > 0 {
		parts = append(parts, fmt.Sprintf("近距≤%d", opts.FarGap))
	}
	if opts.OcrInSec != 0 {
		parts = append(parts, fmt.Sprintf("OCR %ds", max(0, opts.OcrInSec)))
	}
	if opts.FarWaitSec > 0 {
		parts = append(parts, fmt.Sprintf("远距 %ds", opts.FarWaitSec))
	}
	if opts.Retry > 0 {
		parts = append(parts, fmt.Sprintf("重试%d", opts.Retry))
	}
	if opts.CfgHint != "" {
		parts = append(parts, opts.CfgHint)
	}
	if opts.Extra != "" {
		parts = append(parts, opts.Extra)
	}
	publish(Update{Phase: PhaseTask, Task: "矿山勘查", Text: strings.Join(parts, " · ")})
}

// MineMining 矿山开采 HUD 字段（0 表示未提供）。
type MineMining struct {
	State                           string
	Selected, Quota, BusySec, Retry int
	Extra                           string
}

// SetMineMining 矿山开采状态发布（对应 Lua StatusHud.setMineMining）。
func SetMineMining(opts MineMining) {
	parts := []string{"矿山开采"}
	if s := labelOf(mineMiningStateLabel, opts.State); s != "" {
		parts = append(parts, s)
	}
	if opts.Selected > 0 && opts.Quota > 0 {
		parts = append(parts, fmt.Sprintf("选卡 %d/%d", opts.Selected, opts.Quota))
	} else if opts.Quota > 0 {
		parts = append(parts, fmt.Sprintf("上限 %d", opts.Quota))
	}
	if opts.BusySec > 0 {
		parts = append(parts, fmt.Sprintf("busy %ds", opts.BusySec))
	}
	if opts.Retry > 0 {
		parts = append(parts, fmt.Sprintf("重试%d", opts.Retry))
	}
	if opts.Extra != "" {
		parts = append(parts, opts.Extra)
	}
	publish(Update{Phase: PhaseTask, Task: "矿山开采", Text: strings.Join(parts, " · ")})
}

// MineBattle 矿山战斗 HUD 字段（0 表示未提供）。
type MineBattle struct {
	State string
	Retry int
	Extra string
}

// SetMineBattle 矿山战斗状态发布（对应 Lua StatusHud.setMineBattle）。
func SetMineBattle(opts MineBattle) {
	parts := []string{"矿山战斗"}
	if s := labelOf(mineBattleStateLabel, opts.State); s != "" {
		parts = append(parts, s)
	}
	if opts.Retry > 0 {
		parts = append(parts, fmt.Sprintf("重试%d", opts.Retry))
	}
	if opts.Extra != "" {
		parts = append(parts, opts.Extra)
	}
	publish(Update{Phase: PhaseTask, Task: "矿山战斗", Text: strings.Join(parts, " · ")})
}

// MineWait 矿山等待 HUD 字段（0 表示未在等待）。
type MineWait struct {
	SurveySec, MiningSec, MarketSec int
	Extra                           string
}

// SetMineWait 各任务等待状态精简显示（对应 Lua StatusHud.setMineWait）。
func SetMineWait(opts MineWait) {
	parts := []string{"等待"}
	if opts.SurveySec > 0 {
		parts = append(parts, fmt.Sprintf("勘查 %ds", opts.SurveySec))
	}
	if opts.MiningSec > 0 {
		parts = append(parts, fmt.Sprintf("开采 %ds", opts.MiningSec))
	}
	if opts.MarketSec > 0 {
		parts = append(parts, fmt.Sprintf("海滩 %ds", opts.MarketSec))
	}
	if opts.Extra != "" {
		parts = append(parts, opts.Extra)
	}
	if len(parts) == 1 {
		parts = append(parts, "任务")
	}
	publish(Update{Phase: PhaseTask, Task: "等待", Text: strings.Join(parts, " · ")})
}

func labelOf(labels map[string]string, state string) string {
	if s, ok := labels[state]; ok {
		return s
	}
	return state
}
