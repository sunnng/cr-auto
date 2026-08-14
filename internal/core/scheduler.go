package core

import (
	"errors"
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
}

// IdleProvider 空闲等待提供者：返回距离下次可运行的等待秒数与 HUD 文本。
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

// Add 注册任务。
func (s *Scheduler) Add(name string, condition func() bool, action func() error) {
	s.tasks = append(s.tasks, Task{Name: name, Condition: condition, Action: action})
	logger.Debug(schedulerTag, "注册任务: %s", name)
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

// Clear 清空任务与空闲提供者。
func (s *Scheduler) Clear() {
	if len(s.tasks) > 0 {
		logger.Debug(schedulerTag, "清空 %d 个任务", len(s.tasks))
	}
	s.tasks = nil
	s.ClearIdleProviders()
}

// Count 任务数量。
func (s *Scheduler) Count() int { return len(s.tasks) }

// Run 执行一轮：按注册顺序串行执行条件满足的任务。
// stopOnError 时遇 panic 返回 ok=false；返回本轮是否有任务执行。
func (s *Scheduler) Run(stopOnError bool) (hasWork bool, ok bool) {
	ran := 0
	var skipped []string

	for _, task := range s.tasks {
		condOk := true
		func() {
			defer func() {
				if recover() != nil {
					condOk = false
				}
			}()
			if !task.Condition() {
				condOk = false
			}
		}()
		if !condOk {
			skipped = append(skipped, task.Name)
			continue
		}

		hasWork = true
		ran++
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
