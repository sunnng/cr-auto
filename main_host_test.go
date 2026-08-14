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
		return host.rt != nil && host.rt.Scheduler.Count() == 4
	})
	host.mu.Lock()
	scheduler := host.rt.Scheduler
	guard := host.rt.Guard
	host.mu.Unlock()
	// M2a：守卫 1 个（网络联机状态不稳定）+ 矿山任务 4 个。
	if scheduler.Count() != 4 {
		t.Fatalf("M2a register must inject 4 mine tasks, got %d", scheduler.Count())
	}
	if guard.TrapCount() != 1 {
		t.Fatalf("M2a register must inject 1 guard trap, got %d", guard.TrapCount())
	}
	host.stop()
	waitFor(t, func() bool { return !host.isRunning() })
}

func (h *Host) isRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}
