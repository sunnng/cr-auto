package status

import (
	"strings"
	"testing"
)

func collectSink() (Sink, *[]Update) {
	var updates []Update
	return func(u Update) { updates = append(updates, u) }, &updates
}

func TestSetMineSurveyLine(t *testing.T) {
	prev := SetSink(func(u Update) {})
	defer SetSink(prev)
	sink, updates := collectSink()
	SetSink(sink)

	SetMineSurvey(MineSurvey{
		State: "running", Floor: 4, Target: 6, Gap: 2, FarGap: 2, Retry: 1, Extra: "OCR 读层…",
	})
	if len(*updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(*updates))
	}
	u := (*updates)[0]
	if u.Task != "矿山勘查" || u.Phase != PhaseTask {
		t.Fatalf("unexpected update: %+v", u)
	}
	want := "矿山勘查 · 读层 · 层4→6 差2 · 近距≤2 · 重试1 · OCR 读层…"
	if u.Text != want {
		t.Fatalf("line=%q want %q", u.Text, want)
	}
}

func TestSetMineSurveyTargetOnly(t *testing.T) {
	prev := SetSink(func(u Update) {})
	defer SetSink(prev)
	sink, updates := collectSink()
	SetSink(sink)

	SetMineSurvey(MineSurvey{Target: 6, FarGap: 2, CfgHint: "轮询60s 远距600s", Extra: "任务启动"})
	u := (*updates)[0]
	want := "矿山勘查 · 目标6层 · 近距≤2 · 轮询60s 远距600s · 任务启动"
	if u.Text != want {
		t.Fatalf("line=%q want %q", u.Text, want)
	}
}

func TestSetMineMiningLine(t *testing.T) {
	prev := SetSink(func(u Update) {})
	defer SetSink(prev)
	sink, updates := collectSink()
	SetSink(sink)

	SetMineMining(MineMining{State: "selectFlow", Selected: 3, Quota: 6, Extra: "选卡 3/6"})
	u := (*updates)[0]
	if u.Task != "矿山开采" || !strings.Contains(u.Text, "选卡 3/6") {
		t.Fatalf("unexpected update: %+v", u)
	}
}

func TestSetMineBattleLine(t *testing.T) {
	prev := SetSink(func(u Update) {})
	defer SetSink(prev)
	sink, updates := collectSink()
	SetSink(sink)

	SetMineBattle(MineBattle{State: "battleLoop", Retry: 2, Extra: "扫描快转…"})
	u := (*updates)[0]
	if u.Task != "矿山战斗" || u.Text != "矿山战斗 · 扫描 · 重试2 · 扫描快转…" {
		t.Fatalf("unexpected update: %+v", u)
	}
}

func TestSetMineWaitLine(t *testing.T) {
	prev := SetSink(func(u Update) {})
	defer SetSink(prev)
	sink, updates := collectSink()
	SetSink(sink)

	SetMineWait(MineWait{SurveySec: 300, MiningSec: 120, Extra: "调度等待"})
	u := (*updates)[0]
	want := "等待 · 勘查 300s · 开采 120s · 调度等待"
	if u.Text != want {
		t.Fatalf("line=%q want %q", u.Text, want)
	}

	*updates = nil
	SetMineWait(MineWait{})
	if len(*updates) != 1 || (*updates)[0].Text != "等待 · 任务" {
		t.Fatalf("empty wait line=%+v", *updates)
	}
}
