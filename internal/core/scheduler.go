package core

import (
	"errors"
	"sort"
	"time"

	"app/internal/lib/logger"
	"app/internal/lib/status"
)

const schedulerTag = "[Scheduler]"

// ErrSkip 任务动作返回 false（本轮结束但非异常，如“离开广场失败跳过”）。
var ErrSkip = errors.New("core: 任务返回 false")

// Task 调度任务：开关（condition）+ 执行体（action）。
type Task struct {
	Name      string
	Condition func() bool
	Action    func() error
	Priority  int // 优先级，越大越先执行；同级保持注册顺序
	MaxRuns   int // 每轮最多执行次数；0 表示每轮一次（Lua 行为）
}

// TaskPolicy 任务执行策略（优先级排序与单轮次数上限）。
// MaxRuns: 每轮最多执行次数；0/1 表示每轮一次（Lua 行为），>1 时每轮连续执行
// 并在每次执行前重新求值条件（计数在每轮开始时清零）。
type TaskPolicy struct {
	Priority int
	MaxRuns  int
}

// IdleProvider 空闲等待提供者：返回距离下次可运行的等待秒数与等待提示文本。
type IdleProvider func() (remainSec int, label string)

// Scheduler 对应 Lua 工程的 core/scheduler.lua：条件任务串行执行。
type Scheduler struct {
	tasks         []Task
	idleProviders map[string]IdleProvider
	now           func() time.Time
}

// NewScheduler 创建空调度器。
func NewScheduler() *Scheduler {
	return &Scheduler{
		idleProviders: map[string]IdleProvider{},
		now:           time.Now,
	}
}

// SetNow 注入时钟（测试用）。
func (s *Scheduler) SetNow(fn func() time.Time) { s.now = fn }

// Add 注册任务（无策略：优先级按注册顺序、每轮执行一次）。
func (s *Scheduler) Add(name string, condition func() bool, action func() error) {
	s.AddWithPolicy(name, TaskPolicy{}, condition, action)
}

// AddWithPolicy 注册带执行策略的任务（面板“优先级/单次上限”接线）。
func (s *Scheduler) AddWithPolicy(name string, policy TaskPolicy, condition func() bool, action func() error) {
	s.tasks = append(s.tasks, Task{
		Name:      name,
		Condition: condition,
		Action:    action,
		Priority:  policy.Priority,
		MaxRuns:   policy.MaxRuns,
	})
	logger.Debug(schedulerTag, "注册任务: %s (priority=%d maxRuns=%d)", name, policy.Priority, policy.MaxRuns)
}

// AddIdleProvider 注册空闲等待提供者。
func (s *Scheduler) AddIdleProvider(name string, provider IdleProvider) {
	s.idleProviders[name] = provider
	logger.Debug(schedulerTag, "注册 idle provider: %s", name)
}

// RemoveIdleProvider 移除空闲等待提供者。
func (s *Scheduler) RemoveIdleProvider(name string) { delete(s.idleProviders, name) }

// GetIdleProviders 返回全部空闲等待提供者。
func (s *Scheduler) GetIdleProviders() map[string]IdleProvider { return s.idleProviders }

// ClearIdleProviders 清空全部空闲等待提供者。
func (s *Scheduler) ClearIdleProviders() { s.idleProviders = map[string]IdleProvider{} }

// Clear 清空任务与空闲提供者（新会话从零开始）。
func (s *Scheduler) Clear() {
	if len(s.tasks) > 0 {
		logger.Debug(schedulerTag, "清空 %d 个任务", len(s.tasks))
	}
	s.tasks = nil
	s.ClearIdleProviders()
}

// Count 任务数量。
func (s *Scheduler) Count() int { return len(s.tasks) }

// Names 按注册顺序返回全部任务名（测试/诊断用）。
func (s *Scheduler) Names() []string {
	names := make([]string, 0, len(s.tasks))
	for _, t := range s.tasks {
		names = append(names, t.Name)
	}
	return names
}

// Tasks 按注册顺序返回全部任务（测试/诊断用）。
func (s *Scheduler) Tasks() []Task {
	return append([]Task(nil), s.tasks...)
}

// Run 执行一轮：按优先级降序（同级保持注册顺序）串行执行条件满足的任务。
// MaxRuns>0 的任务每轮最多执行该次数，且连续执行之间重新求值条件。
// stopOnError 时遇 panic 返回 ok=false；返回本轮是否有任务执行。
func (s *Scheduler) Run(stopOnError bool) (hasWork bool, ok bool) {
	ran := 0
	var skipped []string

	for _, task := range s.orderedByPriority() {
		runsThisRound := 0
		for {
			condOk := true
			panicked := false
			func() {
				defer func() {
					if recover() != nil {
						condOk = false
						panicked = true
					}
				}()
				if !task.Condition() {
					condOk = false
				}
			}()
			if !condOk {
				if panicked {
					logger.Warn(schedulerTag, "[条件] %s 异常，本轮跳过", task.Name)
				}
				skipped = append(skipped, task.Name)
				break
			}

			hasWork = true
			ran++
			runsThisRound++
			status.SetTask(task.Name, "…")
			logger.Info(schedulerTag, "[执行] %s 开始", task.Name)

			start := s.now()
			err := safeCallError(task.Action)
			elapsed := s.now().Sub(start)

			if err != nil {
				if errors.Is(err, ErrSkip) {
					logger.Warn(schedulerTag, "[执行] %s 结束 false (%.1fs)", task.Name, elapsed.Seconds())
				} else {
					logger.Error(schedulerTag, "[执行] %s 异常 (%.1fs) | %v", task.Name, elapsed.Seconds(), err)
					if stopOnError {
						status.SetIdle()
						return hasWork, false
					}
				}
			} else {
				logger.Info(schedulerTag, "[执行] %s 完成 (%.1fs)", task.Name, elapsed.Seconds())
			}
			if task.MaxRuns <= 1 || runsThisRound >= task.MaxRuns {
				// 每轮一次（MaxRuns 0/1）或单轮上限已到；下一轮重新计数。
				break
			}
		}
	}

	if ran == 0 {
		if len(skipped) > 0 {
			logger.Debug(schedulerTag, "[轮次] 无任务执行 (跳过:%v)", skipped)
		} else {
			logger.Debug(schedulerTag, "[轮次] 无任务执行")
		}
	} else {
		logger.Debug(schedulerTag, "[轮次] 执行 %d 个任务", ran)
	}
	return hasWork, true
}

// SetPolicy 更新已注册任务的执行策略（面板保存后会话内生效）。
// 任务不存在时返回 false。
func (s *Scheduler) SetPolicy(name string, policy TaskPolicy) bool {
	for i := range s.tasks {
		if s.tasks[i].Name == name {
			s.tasks[i].Priority = policy.Priority
			if policy.MaxRuns > 0 {
				s.tasks[i].MaxRuns = policy.MaxRuns
			}
			return true
		}
	}
	return false
}

// orderedByPriority 本轮执行顺序：优先级降序，同级保持注册顺序（稳定排序）。
func (s *Scheduler) orderedByPriority() []Task {
	ordered := append([]Task(nil), s.tasks...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Priority > ordered[j].Priority })
	return ordered
}

// safeCallError 执行 fn 并把 panic 转为错误。
func safeCallError(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("core: panic: " + anyString(r))
		}
	}()
	return fn()
}

func anyString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "unknown panic"
}
