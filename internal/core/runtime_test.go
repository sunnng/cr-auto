package core

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"app/internal/lib/color"
	"app/internal/lib/status"
	"app/internal/vision"
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

func TestRuntimeRoundHookReceivesRoundAndHasWork(t *testing.T) {
	sched := NewScheduler()
	var rounds []struct {
		round   int
		hasWork bool
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt := newTestRuntime(sched, NewGuard())
	rt.RoundHook = func(round int, hasWork bool) {
		rounds = append(rounds, struct {
			round   int
			hasWork bool
		}{round, hasWork})
		if round >= 2 {
			cancel()
		}
	}
	rt.Register = func() {
		sched.Add("t1", func() bool { return true }, func() error { return nil })
	}
	if err := rt.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(rounds) < 2 {
		t.Fatalf("round hook must fire per round, got %d rounds", len(rounds))
	}
	if rounds[0].round != 1 || !rounds[0].hasWork {
		t.Fatalf("round 1 must report hasWork=true: %+v", rounds[0])
	}
}

func TestRuntimeRoundHookCancelStopsPromptly(t *testing.T) {
	sched := NewScheduler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt := newTestRuntime(sched, NewGuard())
	cancelled := false
	rt.RoundHook = func(round int, hasWork bool) {
		if round == 1 {
			cancelled = true
			cancel()
		}
	}
	started := time.Now()
	if err := rt.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !cancelled || time.Since(started) > 2*time.Second {
		t.Fatalf("hook cancel must stop the loop promptly: cancelled=%v elapsed=%v", cancelled, time.Since(started))
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

func TestRuntimeWiresGuardHookIntoColorWait(t *testing.T) {
	popup := vision.Feature{Points: "1|1|ff0000-000000"}
	waitFeat := vision.Feature{Points: "2|2|00ff00-000000"}
	popupColors := vision.DetectsColors(popup)
	waitColors := vision.DetectsColors(waitFeat)
	var n atomic.Int32
	s := color.NewScriptedScreen()
	s.DetectsFn = func(colors string, sim float32) bool {
		cur := n.Add(1)
		if colors == waitColors {
			return false
		}
		if colors == popupColors {
			return cur >= 3
		}
		return false
	}
	color.SetScreen(s)
	color.SetSleep(func(ms int) {})
	defer func() {
		color.SetScreen(nil)
		color.SetSleep(nil)
	}()

	sched := NewScheduler()
	guard := NewGuard()
	handled := 0

	rt := newTestRuntime(sched, guard)
	rt.Register = func() {
		guard.Register("弹窗", vision.Feature{Points: "1|1|ff0000-000000"},
			func() { handled++ }, 10)
		sched.Add("waitTask", func() bool { return true }, func() error {
			// 永不命中的特征 → 进入 wait 轮询，期间依赖守卫钩子清弹窗。
			color.Wait(vision.Feature{Points: "2|2|00ff00-000000"}, 500, 5)
			return nil
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	go rt.Run(ctx)
	defer cancel()

	waitFor(t, func() bool { return handled >= 1 })
	cancel()
	waitDone(t, rt)

	// 停止后守卫钩子必须注销。
	before := handled
	color.TickGuard()
	if handled != before {
		t.Fatal("guard hook must be unregistered after runtime stops")
	}
}
