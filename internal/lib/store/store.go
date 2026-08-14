package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Store 对应 Lua 工程的 lib/store.lua：store.json 键值读写（会话进度、用户配置等）。
// 路径可注入，桌面测试直接写临时目录。
type Store struct {
	mu    sync.Mutex
	path  string
	cache map[string]any
}

// New 创建指向 path 的键值存储。首次读写时加载，写入时落盘。
func New(path string) *Store {
	return &Store{path: path}
}

var defaultStore *Store

// SetDefault 设置包级默认存储（设备端由 main 注入；会话与用户配置共用）。
func SetDefault(s *Store) { defaultStore = s }

// Default 返回包级默认存储；未设置时返回空存储（读写不落盘）。
func Default() *Store {
	if defaultStore != nil {
		return defaultStore
	}
	return &Store{}
}

func (s *Store) load() (map[string]any, error) {
	if s.cache != nil {
		return s.cache, nil
	}
	data := map[string]any{}
	raw, err := os.ReadFile(s.path)
	if err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			// 损坏文件按 Lua store.lua 语义重置为空：下次写入时覆盖，不阻塞读写。
			data = map[string]any{}
		}
	}
	s.cache = data
	return s.cache, nil
}

func (s *Store) save() error {
	if s.cache == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(s.cache)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

// Get 读取键值，键不存在时返回 default。
func (s *Store) Get(key string, def any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return def, err
	}
	if v, ok := data[key]; ok {
		return v, nil
	}
	return def, nil
}

// Set 写入键值并落盘。
func (s *Store) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.load(); err != nil {
		return err
	}
	s.cache[key] = value
	return s.save()
}

// Del 删除键并落盘。
func (s *Store) Del(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.load(); err != nil {
		return err
	}
	delete(s.cache, key)
	return s.save()
}

// Has 键是否存在（值为 nil 视为不存在）。
func (s *Store) Has(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return false, err
	}
	v, ok := data[key]
	return ok && v != nil, nil
}

// Incr 数值自增/自减（键不存在时从 def 起算），返回新值。
func (s *Store) Incr(key string, delta, def int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return 0, err
	}
	current := def
	switch v := data[key].(type) {
	case float64:
		current = int(v)
	case int:
		current = v
	}
	current += delta
	data[key] = current
	if err := s.save(); err != nil {
		return 0, err
	}
	return current, nil
}

// Clear 清空存储并落盘。
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = map[string]any{}
	return s.save()
}
