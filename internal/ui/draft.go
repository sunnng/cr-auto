// The panel's local settings draft. It holds the UI-editable configuration
// surface: the field groups the ImGui renderer reads and edits (Run, Safety,
// Tasks) plus field groups reserved for future pages (Production, HUD,
// Plans). Persistence, schema migration and time-window rules are host
// responsibilities that arrive through the command seam.
package ui

import (
	"errors"
	"fmt"
)

type RunMode string

const (
	RunManual    RunMode = "manual"
	RunOnce      RunMode = "once"
	RunScheduled RunMode = "scheduled"
)

type HUDMode string

const (
	HUDCompact  HUDMode = "compact"
	HUDDetailed HUDMode = "detailed"
)

type HUDPosition string

const (
	HUDTopLeft     HUDPosition = "top-left"
	HUDTopRight    HUDPosition = "top-right"
	HUDBottomLeft  HUDPosition = "bottom-left"
	HUDBottomRight HUDPosition = "bottom-right"
)

type Draft struct {
	Run        RunSettings
	Safety     SafetySettings
	Production ProductionSettings
	HUD        HUDSettings
	Tasks      map[string]TaskSetting
	Plans      []CustomPlan
}

type RunSettings struct {
	Mode        RunMode
	StartMinute int
	EndMinute   int
}

type SafetySettings struct {
	MinConfidence     float32
	MaxActionsPerRun  int
	UnknownTimeoutSec int
	// BlockResourceSpend and StopOnSensitivePage are always forced on by the
	// safety page; the fields remain so the rendered controls match the draft.
	BlockResourceSpend  bool
	StopOnSensitivePage bool
}

type ProductionSettings struct {
	AllowCoinAndMaterialSpend bool
}

type HUDSettings struct {
	Enabled  bool
	Mode     HUDMode
	Position HUDPosition
	Opacity  int
	TextSize int
}

type TaskSetting struct {
	Enabled  bool
	Priority int
	MaxRuns  int
}

type CustomPlan struct {
	ID      string
	Name    string
	TaskIDs []string
	Repeat  int
}

func Default() Draft {
	return Draft{
		Run: RunSettings{
			Mode:        RunManual,
			StartMinute: 0,
			EndMinute:   1439,
		},
		Safety: SafetySettings{
			MinConfidence:       0.92,
			MaxActionsPerRun:    300,
			UnknownTimeoutSec:   30,
			BlockResourceSpend:  true,
			StopOnSensitivePage: true,
		},
		Production: ProductionSettings{AllowCoinAndMaterialSpend: true},
		HUD: HUDSettings{
			Enabled:  true,
			Mode:     HUDCompact,
			Position: HUDTopLeft,
			Opacity:  72,
			TextSize: 28,
		},
		Tasks: map[string]TaskSetting{},
		Plans: []CustomPlan{},
	}
}

func (d Draft) Validate() error {
	if d.Run.StartMinute < 0 || d.Run.StartMinute > 1439 || d.Run.EndMinute < 0 || d.Run.EndMinute > 1439 {
		return errors.New("运行时段必须在 0..1439 分钟内")
	}
	if d.Run.EndMinute < d.Run.StartMinute {
		return errors.New("结束分钟不能早于开始分钟")
	}
	if d.Safety.MinConfidence < 0.90 || d.Safety.MinConfidence > 0.99 {
		return fmt.Errorf("最低视觉置信度必须在 0.90..0.99 之间，当前 %.2f", d.Safety.MinConfidence)
	}
	if d.Safety.MaxActionsPerRun < 1 || d.Safety.MaxActionsPerRun > 1000 {
		return fmt.Errorf("单次动作预算必须在 1..1000 之间，当前 %d", d.Safety.MaxActionsPerRun)
	}
	if d.Safety.UnknownTimeoutSec < 5 || d.Safety.UnknownTimeoutSec > 300 {
		return fmt.Errorf("未知场景超时必须在 5..300 秒之间，当前 %d", d.Safety.UnknownTimeoutSec)
	}
	return nil
}

func cloneDraft(src Draft) Draft {
	dst := src
	dst.Tasks = make(map[string]TaskSetting, len(src.Tasks))
	for id, task := range src.Tasks {
		dst.Tasks[id] = task
	}
	dst.Plans = make([]CustomPlan, len(src.Plans))
	for i, plan := range src.Plans {
		dst.Plans[i] = plan
		dst.Plans[i].TaskIDs = append([]string(nil), plan.TaskIDs...)
	}
	return dst
}
