package status

import "testing"

func recorder(t *testing.T) (func() []Update, func()) {
	t.Helper()
	var updates []Update
	prev := sink
	SetSink(func(u Update) { updates = append(updates, u) })
	restore := func() { SetSink(prev) }
	return func() []Update { return updates }, restore
}

func TestPublishPhases(t *testing.T) {
	snapshot, restore := recorder(t)
	defer restore()
	Set(PhaseRun, "运行中")
	SetTask("矿山勘查", "远距等待 600s")
	SetWait("勘查 600s · 开采 1200s")
	SetIdle()
	updates := snapshot()
	want := []Update{
		{Phase: PhaseRun, Text: "运行中"},
		{Phase: PhaseTask, Task: "矿山勘查", Text: "远距等待 600s"},
		{Phase: PhaseWait, Text: "勘查 600s · 开采 1200s"},
		{Phase: PhaseIdle},
	}
	if len(updates) != len(want) {
		t.Fatalf("updates=%+v", updates)
	}
	for i := range want {
		if updates[i] != want[i] {
			t.Fatalf("update[%d]=%+v want %+v", i, updates[i], want[i])
		}
	}
}

func TestNilSinkDiscards(t *testing.T) {
	SetSink(nil)
	SetIdle()
}
