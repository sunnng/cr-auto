package core

import (
	"errors"
	"testing"
	"time"
)

func TestInitSetsStateAndResetsCounters(t *testing.T) {
	sm := New()
	sm.Init("start", InitOpts{MaxRetry: 2, TimeoutSec: 100})
	if sm.GetState() != "start" {
		t.Fatalf("state=%s", sm.GetState())
	}
	sm.RetryActive()
	sm.Init("again", InitOpts{})
	if sm.retries != 0 || sm.errors != 0 {
		t.Fatalf("init must reset counters: %d/%d", sm.retries, sm.errors)
	}
}

func TestToResetsCounters(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{})
	sm.RetryActive()
	sm.To("b")
	if sm.GetState() != "b" || sm.retries != 0 {
		t.Fatalf("state=%s retries=%d", sm.GetState(), sm.retries)
	}
}

func TestRetryActiveAndErrorLimits(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{MaxRetry: 1})
	if ok, _ := sm.RetryActive(); !ok {
		t.Fatal("first retry must be allowed")
	}
	if ok, _ := sm.RetryActive(); ok {
		t.Fatal("retry beyond max must fail")
	}
	sm2 := New()
	sm2.Init("a", InitOpts{MaxError: 1})
	sm2.RetryError()
	if ok, _ := sm2.RetryError(); ok {
		t.Fatal("error retry beyond max must fail")
	}
}

func TestRunHappyPath(t *testing.T) {
	sm := New()
	sm.Init("start", InitOpts{})
	var steps []string
	ok, err := sm.Run(map[string]StateHandler{
		"start": func(sm *StateMachine) (string, error) {
			steps = append(steps, "start")
			return "end", nil
		},
		"end": func(sm *StateMachine) (string, error) {
			steps = append(steps, "end")
			return DONE, nil
		},
	}, RunOpts{Interval: 0, Label: "测试"})
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(steps) != 2 || steps[0] != "start" || steps[1] != "end" {
		t.Fatalf("steps=%v", steps)
	}
}

func TestRunKEEPStaysInStateWithoutRetryCount(t *testing.T) {
	sm := New()
	sm.Init("wait", InitOpts{})
	times := 0
	ok, err := sm.Run(map[string]StateHandler{
		"wait": func(sm *StateMachine) (string, error) {
			times++
			if times < 3 {
				return KEEP, nil
			}
			return DONE, nil
		},
	}, RunOpts{Interval: 0, Label: "测试"})
	if !ok || err != nil || times != 3 {
		t.Fatalf("ok=%v err=%v times=%d", ok, err, times)
	}
	if sm.retries != 0 {
		t.Fatalf("KEEP must not count as retry: %d", sm.retries)
	}
}

func TestRunNilReturnTreatedAsKEEP(t *testing.T) {
	sm := New()
	sm.Init("wait", InitOpts{})
	times := 0
	ok, _ := sm.Run(map[string]StateHandler{
		"wait": func(sm *StateMachine) (string, error) {
			times++
			if times == 1 {
				return "", nil
			}
			return DONE, nil
		},
	}, RunOpts{Interval: 0, Label: "测试"})
	if !ok || times != 2 {
		t.Fatalf("ok=%v times=%d", ok, times)
	}
}

func TestRunRETRYCountsActiveRetries(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{MaxRetry: 2})
	times := 0
	ok, err := sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) {
			times++
			if times < 3 {
				return RETRY, nil
			}
			return DONE, nil
		},
	}, RunOpts{Interval: 0, Label: "测试"})
	if !ok || err != nil || times != 3 {
		t.Fatalf("ok=%v err=%v times=%d", ok, err, times)
	}
	if sm.retries != 2 {
		t.Fatalf("retries=%d", sm.retries)
	}
}

func TestRunRETRYExceededTerminates(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{MaxRetry: 1})
	_, err := sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) { return RETRY, nil },
	}, RunOpts{Interval: 0, Label: "测试"})
	if err == nil {
		t.Fatal("retry over limit must terminate with error")
	}
}

func TestRunHandlerPanicTriggersErrorRetry(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{MaxError: 1})
	times := 0
	ok, err := sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) {
			times++
			if times < 2 {
				panic("boom")
			}
			return DONE, nil
		},
	}, RunOpts{Interval: 0, Label: "测试"})
	if !ok || err != nil || times != 2 {
		t.Fatalf("ok=%v err=%v times=%d", ok, err, times)
	}
	if sm.errors != 1 {
		t.Fatalf("errors=%d", sm.errors)
	}
}

func TestRunFatalErrorTerminates(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{})
	_, err := sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) { return "", errors.New("致命") },
	}, RunOpts{Interval: 0, Label: "测试"})
	if err == nil || err.Error() != "致命" {
		t.Fatalf("err=%v", err)
	}
}

func TestRunUnknownStateTerminates(t *testing.T) {
	sm := New()
	sm.Init("missing", InitOpts{})
	_, err := sm.Run(map[string]StateHandler{}, RunOpts{Interval: 0, Label: "测试"})
	if err == nil {
		t.Fatal("unknown state must terminate")
	}
}

func TestRunTimeout(t *testing.T) {
	sm := New()
	now := time.Unix(1000, 0)
	sm.SetNow(func() time.Time { return now })
	sm.SetSleep(func(ms int) {
		now = now.Add(time.Duration(ms) * time.Millisecond)
	})
	sm.Init("a", InitOpts{TimeoutSec: 1})
	_, err := sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) { return KEEP, nil },
	}, RunOpts{Interval: 200, Label: "测试"})
	if err == nil {
		t.Fatal("timeout must terminate")
	}
}

func TestRunGuardCalledEachTick(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{})
	guardRuns := 0
	handlerTicks := 0
	ok, err := sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) {
			handlerTicks++
			if handlerTicks < 3 {
				return KEEP, nil
			}
			return DONE, nil
		},
	}, RunOpts{Interval: 0, Guard: func() { guardRuns++ }, Label: "测试"})
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if guardRuns < handlerTicks {
		t.Fatalf("guard must run each tick: guard=%d handlerTicks=%d", guardRuns, handlerTicks)
	}
}

func TestRunRetryIntervalSleepUsed(t *testing.T) {
	sm := New()
	sm.Init("a", InitOpts{RetryIntervalMs: 1000})
	var sleeps []int
	sm.SetSleep(func(ms int) { sleeps = append(sleeps, ms) })
	_, _ = sm.Run(map[string]StateHandler{
		"a": func(sm *StateMachine) (string, error) { return RETRY, nil },
	}, RunOpts{Interval: 200, Label: "测试"})
	// 每次 RETRY 后 sleep 1000ms（按 interval 分片）；maxRetry=3 允许 3 次重试后终止。
	total := 0
	for _, s := range sleeps {
		total += s
	}
	if total != 3000 {
		t.Fatalf("retry sleeps must total 3000ms, got %v (sum=%d)", sleeps, total)
	}
}
