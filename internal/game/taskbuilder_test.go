package game

import (
	"path/filepath"
	"strconv"
	"testing"

	"app/internal/core"
	"app/internal/lib/store"
	"app/internal/lib/userconfig"
)

func newTestUserConfig(t *testing.T) *userconfig.UserConfig {
	t.Helper()
	return userconfig.New(store.New(filepath.Join(t.TempDir(), "store.json")))
}

func TestTaskBuilderConfigKeyGate(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	if err := uc.Set("arena", map[string]any{"Enabled": true}); err != nil {
		t.Fatal(err)
	}
	ran := false
	NewTask(s, uc, "竞技场", TaskOptions{
		ConfigKey: "arena",
		Action:    func() error { ran = true; return nil },
	})
	hasWork, ok := s.Run(false)
	if !ok || !hasWork || !ran {
		t.Fatalf("configKey enabled must run: hasWork=%v ok=%v ran=%v", hasWork, ok, ran)
	}
}

func TestTaskBuilderConfigKeyDisabledSkips(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	NewTask(s, uc, "竞技场", TaskOptions{
		ConfigKey: "arena",
		Action:    func() error { ran = true; return nil },
	})
	hasWork, ok := s.Run(false)
	if !ok || hasWork || ran {
		t.Fatalf("disabled task must skip: hasWork=%v ok=%v ran=%v", hasWork, ok, ran)
	}
}

func TestTaskBuilderCustomCheckEnabledWins(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	NewTask(s, uc, "任务", TaskOptions{
		ConfigKey:    "arena",
		CheckEnabled: func() bool { return true },
		Action:       func() error { ran = true; return nil },
	})
	s.Run(false)
	if !ran {
		t.Fatal("custom checkEnabled must override configKey")
	}
}

func TestTaskBuilderPreconditionBlocks(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	blocked := false
	NewTask(s, uc, "任务", TaskOptions{
		CheckEnabled:       func() bool { return true },
		Precondition:       func() bool { return false },
		OnPreconditionFail: func() { blocked = true },
		Action:             func() error { ran = true; return nil },
	})
	hasWork, ok := s.Run(false)
	if !ok || hasWork || ran || !blocked {
		t.Fatalf("precondition must block: hasWork=%v ok=%v ran=%v blocked=%v", hasWork, ok, ran, blocked)
	}
}

func TestTaskBuilderCanResumeSkipsCheckReady(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	readyChecked := false
	NewTask(s, uc, "任务", TaskOptions{
		CheckEnabled: func() bool { return true },
		CanResume:    func() bool { return true },
		CheckReady: func() (bool, int) {
			readyChecked = true
			return false, 600
		},
		Action: func() error { ran = true; return nil },
	})
	s.Run(false)
	if !ran || readyChecked {
		t.Fatalf("canResume must skip checkReady: ran=%v readyChecked=%v", ran, readyChecked)
	}
}

func TestTaskBuilderCheckReadyNotReadyShowsHud(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	notReadyRemain := 0
	NewTask(s, uc, "矿山勘查", TaskOptions{
		CheckEnabled: func() bool { return true },
		CheckReady:   func() (bool, int) { return false, 600 },
		WaitHud:      func(remain int) string { return "远距等待 " + strconv.Itoa(remain) + "s" },
		OnNotReady:   func(remain int) { notReadyRemain = remain },
		Action:       func() error { ran = true; return nil },
	})
	hasWork, ok := s.Run(false)
	if !ok || hasWork || ran {
		t.Fatalf("not-ready task must skip: hasWork=%v ok=%v ran=%v", hasWork, ok, ran)
	}
	if notReadyRemain != 600 {
		t.Fatalf("onNotReady remain=%d", notReadyRemain)
	}
}

func TestTaskBuilderLeaveSquareFailureSkips(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	NewTask(s, uc, "任务", TaskOptions{
		CheckEnabled: func() bool { return true },
		LeaveSquare:  func() bool { return false },
		Action:       func() error { ran = true; return nil },
	})
	hasWork, ok := s.Run(false)
	if !ok || !hasWork || ran {
		t.Fatalf("leaveSquare failure must skip action but count as work: hasWork=%v ok=%v ran=%v", hasWork, ok, ran)
	}
}

func TestTaskBuilderLeaveSquareSuccessRuns(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	left := false
	NewTask(s, uc, "任务", TaskOptions{
		CheckEnabled: func() bool { return true },
		LeaveSquare:  func() bool { left = true; return true },
		Action:       func() error { ran = true; return nil },
	})
	s.Run(false)
	if !ran || !left {
		t.Fatalf("leaveSquare success must run: ran=%v left=%v", ran, left)
	}
}

func TestTaskBuilderNoLeaveSquareHookRunsDirectly(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	ran := false
	NewTask(s, uc, "任务", TaskOptions{
		CheckEnabled: func() bool { return true },
		Action:       func() error { ran = true; return nil },
	})
	s.Run(false)
	if !ran {
		t.Fatal("task without leaveSquare hook must run directly")
	}
}

func TestTaskBuilderActionPanicReported(t *testing.T) {
	s := core.NewScheduler()
	uc := newTestUserConfig(t)
	NewTask(s, uc, "任务", TaskOptions{
		CheckEnabled: func() bool { return true },
		Action:       func() error { panic("boom") },
	})
	hasWork, ok := s.Run(true)
	if ok || !hasWork {
		t.Fatalf("panic must propagate stopOnError: hasWork=%v ok=%v", hasWork, ok)
	}
}
