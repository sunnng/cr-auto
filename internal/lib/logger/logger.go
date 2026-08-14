// Package logger 对应 Lua 工程的 lib/logger.lua：分级日志，宿主注入输出端。
package logger

import (
	"fmt"
	"sync"
)

// Level 日志级别。
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Sink 日志输出端（设备端由 main 注入 AutoGo utils.LogI/LogE）。
type Sink func(level Level, tag string, message string)

var (
	mu   sync.RWMutex
	now  = LevelInfo // Lua logger.lua 默认 level=3（INFO）
	sink Sink
)

// SetLevel 设置最小输出级别（默认 Info，对应 Lua logger.level=3）。
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	now = level
}

// SetSink 注入输出端；nil 表示丢弃。
func SetSink(fn Sink) {
	mu.Lock()
	defer mu.Unlock()
	sink = fn
}

func log(level Level, tag, format string, args ...any) {
	mu.RLock()
	current := now
	fn := sink
	mu.RUnlock()
	if level < current || fn == nil {
		return
	}
	fn(level, tag, fmt.Sprintf(format, args...))
}

// Debug 调试日志（轮次细节）。
func Debug(tag, format string, args ...any) { log(LevelDebug, tag, format, args...) }

// Info 常规日志。
func Info(tag, format string, args ...any) { log(LevelInfo, tag, format, args...) }

// Warn 警告日志。
func Warn(tag, format string, args ...any) { log(LevelWarn, tag, format, args...) }

// Error 错误日志。
func Error(tag, format string, args ...any) { log(LevelError, tag, format, args...) }
