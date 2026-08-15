package core

import (
	"testing"
)

func TestSchedulerAddAndCount(t *testing.T) {
	s := NewScheduler()
	s.Add("t1", func() bool { return false }, func() error { return nil })
	s.Add("t2", func() bool { return false }, func() error { return nil })
	if s.Count() != 2 {
		t.Fatalf("count=%d", s.Count())
	}
	s.Clear()
	if s.Count() != 0 {
		t.Fatal("clear must empty tasks")
	}
}

func TestSchedulerRunsOnlyReadyTasks(t *testing.T) {
	s := NewScheduler()
	var ran []string
	s.Add("ready", func() bool { return true }, func() error { ran = append(ran, "ready"); return nil })
	s.Add("skip", func() bool { return false }, func() error { ran = append(ran, "skip"); return nil })
	hasWork, ok := s.Run(false)
	if !ok || !hasWork {
		t.Fatalf("hasWork=%v ok=%v", hasWork, ok)
	}
	if len(ran) != 1 || ran[0] != "ready" {
		t.Fatalf("ran=%v", ran)
	}
}

func TestSchedulerNoWork(t *testing.T) {
	s := NewScheduler()
	s.Add("skip", func() bool { return false }, func() error { return nil })
	hasWork, ok := s.Run(false)
	if !ok || hasWork {
		t.Fatalf("no work must keep ok=true: hasWork=%v ok=%v", hasWork, ok)
	}
}

func TestSchedulerStopOnErrorAbortsRound(t *testing.T) {
	s := NewScheduler()
	var ran []string
	s.Add("boom", func() bool { return true }, func() error { panic("kaboom") })
	s.Add("after", func() bool { return true }, func() error { ran = append(ran, "after"); return nil })
	hasWork, ok := s.Run(true)
	if ok || !hasWork {
		t.Fatalf("stopOnError must abort: hasWork=%v ok=%v", hasWork, ok)
	}
	if len(ran) != 0 {
		t.Fatalf("task after panic must not run: %v", ran)
	}
}

func TestSchedulerPanicWithoutStopOnErrorContinues(t *testing.T) {
	s := NewScheduler()
	var ran []string
	s.Add("boom", func() bool { return true }, func() error { panic("kaboom") })
	s.Add("after", func() bool { return true }, func() error { ran = append(ran, "after"); return nil })
	hasWork, ok := s.Run(false)
	if !ok || !hasWork {
		t.Fatalf("hasWork=%v ok=%v", hasWork, ok)
	}
	if len(ran) != 1 || ran[0] != "after" {
		t.Fatalf("must continue after panic: %v", ran)
	}
}

func TestSchedulerErrSkipTreatedAsNotCompleted(t *testing.T) {
	s := NewScheduler()
	ran := false
	s.Add("skipTask", func() bool { return true }, func() error { ran = true; return ErrSkip })
	hasWork, ok := s.Run(false)
	if !ok || !hasWork || !ran {
		t.Fatalf("ErrSkip must count as work: hasWork=%v ok=%v ran=%v", hasWork, ok, ran)
	}
}

func TestSchedulerConditionPanicSkipsTask(t *testing.T) {
	s := NewScheduler()
	ran := false
	s.Add("badCond", func() bool { panic("cond") }, func() error { ran = true; return nil })
	hasWork, ok := s.Run(false)
	if !ok || hasWork || ran {
		t.Fatalf("condition panic must skip task: hasWork=%v ok=%v ran=%v", hasWork, ok, ran)
	}
}

func TestSchedulerRunsInRegistrationOrder(t *testing.T) {
	s := NewScheduler()
	var order []string
	s.Add("first", func() bool { return true }, func() error { order = append(order, "first"); return nil })
	s.Add("second", func() bool { return true }, func() error { order = append(order, "second"); return nil })
	s.Run(false)
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("order=%v", order)
	}
}

func TestSchedulerPriorityOrdersExecution(t *testing.T) {
	s := NewScheduler()
	var order []string
	s.Add("low", func() bool { return true }, func() error { order = append(order, "low"); return nil })
	s.AddWithPolicy("high", TaskPolicy{Priority: 100}, func() bool { return true }, func() error { order = append(order, "high"); return nil })
	s.Run(false)
	if len(order) != 2 || order[0] != "high" || order[1] != "low" {
		t.Fatalf("priority must reorder execution: %v", order)
	}
}

func TestSchedulerPriorityStableKeepsRegistrationOrder(t *testing.T) {
	s := NewScheduler()
	var order []string
	s.Add("a", func() bool { return true }, func() error { order = append(order, "a"); return nil })
	s.AddWithPolicy("b", TaskPolicy{Priority: 0}, func() bool { return true }, func() error { order = append(order, "b"); return nil })
	s.Add("c", func() bool { return true }, func() error { order = append(order, "c"); return nil })
	s.Run(false)
	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("equal priority must keep registration order: %v", order)
	}
}

func TestSchedulerMaxRunsPerRound(t *testing.T) {
	s := NewScheduler()
	ran := 0
	s.AddWithPolicy("chunk", TaskPolicy{MaxRuns: 3}, func() bool { return true }, func() error { ran++; return nil })
	hasWork, ok := s.Run(false)
	if !ok || !hasWork {
		t.Fatalf("hasWork=%v ok=%v", hasWork, ok)
	}
	if ran != 3 {
		t.Fatalf("task must run at most MaxRuns times per round, ran=%d", ran)
	}
	// 上限是每轮而非会话：第二轮重新计数。
	s.Run(false)
	if ran != 6 {
		t.Fatalf("MaxRuns must reset each round, total after 2 rounds=%d", ran)
	}
}

func TestSchedulerMaxRunsRechecksCondition(t *testing.T) {
	s := NewScheduler()
	ran := 0
	ready := true
	s.AddWithPolicy("gated", TaskPolicy{MaxRuns: 5}, func() bool { return ready }, func() error { ran++; ready = false; return nil })
	s.Run(false)
	if ran != 1 {
		t.Fatalf("condition must be re-evaluated between runs, ran=%d", ran)
	}
}

func TestSchedulerMaxRunsZeroRunsOncePerRound(t *testing.T) {
	s := NewScheduler()
	ran := 0
	s.AddWithPolicy("free", TaskPolicy{MaxRuns: 0}, func() bool { return true }, func() error { ran++; return nil })
	hasWork, ok := s.Run(false)
	if !ok || !hasWork {
		t.Fatalf("hasWork=%v ok=%v", hasWork, ok)
	}
	if ran != 1 {
		t.Fatalf("task without MaxRuns must run once per round, ran=%d", ran)
	}
}

func TestSchedulerSetPolicyUpdatesExistingTask(t *testing.T) {
	s := NewScheduler()
	var order []string
	s.Add("first", func() bool { return true }, func() error { order = append(order, "first"); return nil })
	s.Add("second", func() bool { return true }, func() error { order = append(order, "second"); return nil })
	if !s.SetPolicy("second", TaskPolicy{Priority: 100}) {
		t.Fatal("SetPolicy must report the task exists")
	}
	s.Run(false)
	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Fatalf("SetPolicy must reorder execution: %v", order)
	}
	if s.SetPolicy("missing", TaskPolicy{}) {
		t.Fatal("SetPolicy on unknown task must report false")
	}
}

func TestSchedulerSetPolicyMaxRuns(t *testing.T) {
	s := NewScheduler()
	ran := 0
	s.Add("chunk", func() bool { return true }, func() error { ran++; return nil })
	s.SetPolicy("chunk", TaskPolicy{MaxRuns: 2})
	s.Run(false)
	if ran != 2 {
		t.Fatalf("SetPolicy MaxRuns must be enforced, ran=%d", ran)
	}
}

func TestSchedulerIdleProviders(t *testing.T) {
	s := NewScheduler()
	s.AddIdleProvider("mine", func() (int, string) { return 600, "勘查 600s" })
	s.AddIdleProvider("battle", func() (int, string) { return 0, "" })
	if len(s.GetIdleProviders()) != 2 {
		t.Fatalf("providers=%d", len(s.GetIdleProviders()))
	}
	s.RemoveIdleProvider("mine")
	if len(s.GetIdleProviders()) != 1 {
		t.Fatalf("remove failed")
	}
	s.ClearIdleProviders()
	if len(s.GetIdleProviders()) != 0 {
		t.Fatal("clear idle providers failed")
	}
}
