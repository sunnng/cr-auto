package core

import (
	"context"
	"testing"
	"time"

	"app/internal/lib/status"
)

// installStatusSink 记录 status 发布序列。
func installStatusSink(t *testing.T) func() []status.Update {
	t.Helper()
	var updates []status.Update
	prev := status.SetSink(func(u status.Update) { updates = append(updates, u) })
	t.Cleanup(func() { status.SetSink(prev) })
	return func() []status.Update { return updates }
}

func newTestRuntime(sched *Scheduler, guard *Guard) *Runtime {
	rt := NewRuntime(RuntimeOptions{
		Scheduler: sched,
		Guard:     guard,
	})
	rt.Register = func() {}
	rt.StepDelayMS = 20
	rt.IdleDelayMS = 100
	rt.GuardIntervalMS = 10
	return rt
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

func waitDone(t *testing.T, rt *Runtime) {
	t.Helper()
	select {
	case <-rt.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("runtime did not stop")
	}
}

func TestRuntimeRunsRegisteredTaskEachRound(t *testing.T) {
	sched := NewScheduler()
	var runs int
	rt := newTestRuntime(sched, NewGuard())
	rt.Register = func() {
		sched.Add("t1", func() bool { return true }, func() error { runs++; return nil })
	}

	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool { return runs >= 2 })
	cancel()
	waitDone(t, rt)
}

func TestRuntimeRegisterInjectedBeforeFirstRound(t *testing.T) {
	sched := NewScheduler()
	registered := false
	rt := newTestRuntime(sched, NewGuard())
	rt.Register = func() { registered = sched.Count() == 0 }

	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool { return registered })
}

func TestRuntimeGuardScannedEveryRound(t *testing.T) {
	sched := NewScheduler()
	guard := NewGuard()
	guardChecks := 0
	rt := newTestRuntime(sched, guard)
	rt.Register = func() {
		guard.Register("probe", func() bool {
			guardChecks++
			return false
		}, func() {}, 0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool { return guardChecks >= 3 })
}

func TestRuntimeIdleProviderFeedsWaitHud(t *testing.T) {
	sched := NewScheduler()
	rt := newTestRuntime(sched, NewGuard())
	rt.IdleDelayMS = 1000
	rt.Register = func() {
		sched.AddIdleProvider("mine", func() (int, string) { return 3, "勘查 3s" })
	}

	updates := installStatusSink(t)
	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool {
		for _, u := range updates() {
			if u.Phase == status.PhaseWait {
				return true
			}
		}
		return false
	})
}

func TestRuntimeStepDelayBetweenRounds(t *testing.T) {
	sched := NewScheduler()
	rt := newTestRuntime(sched, NewGuard())
	rt.StepDelayMS = 50
	rt.Register = func() {
		sched.Add("t", func() bool { return true }, func() error { return nil })
	}

	var sleeps []int
	rt.Guard.SetSleep(func(ms int) { sleeps = append(sleeps, ms) })

	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool { return len(sleeps) >= 10 })
	// Guard.Sleep 按 GuardIntervalMS 分片：50ms 步进等待 = 5×10ms。
	for _, s := range sleeps {
		if s != 10 {
			t.Fatalf("step sleep must fragment at GuardIntervalMS: %v", sleeps)
		}
	}
}

func TestRuntimeStopOnErrorTerminates(t *testing.T) {
	sched := NewScheduler()
	rt := newTestRuntime(sched, NewGuard())
	rt.StopOnError = true
	rt.Register = func() {
		sched.Add("boom", func() bool { return true }, func() error { panic("kaboom") })
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := rt.Run(ctx); err == nil {
		t.Fatal("stopOnError must abort the runtime")
	}
}

func TestRuntimePauseHaltsAndResumeContinues(t *testing.T) {
	sched := NewScheduler()
	var runs int
	rt := newTestRuntime(sched, NewGuard())
	rt.StepDelayMS = 20
	rt.Register = func() {
		sched.Add("t", func() bool { return true }, func() error { runs++; return nil })
	}

	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool { return runs >= 2 })
	rt.Pause()
	time.Sleep(120 * time.Millisecond)
	frozen := runs
	time.Sleep(120 * time.Millisecond)
	if runs != frozen {
		t.Fatalf("paused runtime must not run tasks: before=%d after=%d", frozen, runs)
	}
	rt.Resume()
	waitFor(t, func() bool { return runs > frozen })
}

func TestRuntimeCancelStopsLoop(t *testing.T) {
	sched := NewScheduler()
	rt := newTestRuntime(sched, NewGuard())
	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	cancel()
	waitDone(t, rt)
}
