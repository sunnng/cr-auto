package main

import (
	"testing"
	"time"

	"app/internal/ui"
)

func openTestPanel(t *testing.T) *ui.Panel {
	t.Helper()
	panel := ui.NewPanel()
	if err := panel.Open(ui.Snapshot{Settings: ui.Default()}, func(ui.Command) {}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(panel.Close)
	return panel
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !cond() {
		t.Fatal("condition not reached in time")
	}
}

func TestHostStartRunsAndStopHalts(t *testing.T) {
	panel := openTestPanel(t)
	host := NewHost(panel)

	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	if !host.stop() {
		t.Fatal("stop must halt a running engine")
	}
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostStartIsIdempotent(t *testing.T) {
	panel := openTestPanel(t)
	host := NewHost(panel)

	host.Handle(ui.Command{Type: ui.CommandStart})
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	rts := 0
	host.mu.Lock()
	if host.rt != nil {
		rts = 1
	}
	host.mu.Unlock()
	if rts != 1 {
		t.Fatal("second start must not create another runtime")
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostPauseResume(t *testing.T) {
	panel := openTestPanel(t)
	host := NewHost(panel)

	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	host.Handle(ui.Command{Type: ui.CommandPause})
	host.Handle(ui.Command{Type: ui.CommandResume})
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func TestHostStopWithoutEngineIsNoop(t *testing.T) {
	panel := openTestPanel(t)
	host := NewHost(panel)
	if host.stop() {
		t.Fatal("stop without engine must report false")
	}
}

func TestHostEngineStartsWithRegisterInjection(t *testing.T) {
	panel := openTestPanel(t)
	host := NewHost(panel)
	host.Handle(ui.Command{Type: ui.CommandStart})
	waitFor(t, func() bool { return host.isRunning() })

	// 注册在 Runtime.Run 内完成，轮询等待注入结果。
	waitFor(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.rt != nil && host.rt.Scheduler.Count() == 9
	})
	host.mu.Lock()
	scheduler := host.rt.Scheduler
	guard := host.rt.Guard
	host.mu.Unlock()
	// M2b：守卫 1 个（网络联机状态不稳定）+ 业务任务 9 个
	// （矿山勘查/开采/战斗/解除洋菜冻 + 海滩交易所/王国竞技场/梦幻繁星岛/布谷鸟广场/洗脆饼词条）。
	if scheduler.Count() != 9 {
		t.Fatalf("M2b register must inject 9 tasks, got %d", scheduler.Count())
	}
	if guard.TrapCount() != 1 {
		t.Fatalf("M2b register must inject 1 guard trap, got %d", guard.TrapCount())
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func (h *Host) isRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}
