package userconfig

import (
	"encoding/json"
	"errors"
	"sync"

	"app/internal/config"
	"app/internal/lib/store"
)

// 对应 Lua 工程的 lib/user-config.lua：默认值(config.User) + 持久化(store)，按模块名读写。
type UserConfig struct {
	mu    sync.Mutex
	st    *store.Store
	cache map[string]map[string]any
}

const storeKey = "user_config"

var errUnknownSection = errors.New("userconfig: 未知配置段")

// defaultsMap 由 config.Static.User 的 JSON 形态生成：section -> 字段默认值。
var defaultsMap = func() map[string]map[string]any {
	raw, err := json.Marshal(config.Static.User)
	if err != nil {
		panic(err)
	}
	var sections map[string]map[string]any
	if err := json.Unmarshal(raw, &sections); err != nil {
		panic(err)
	}
	return sections
}()

// New 创建基于 st 持久化的用户配置。
func New(st *store.Store) *UserConfig {
	return &UserConfig{st: st}
}

// Default 返回基于包级默认存储的配置（设备端 main 注入 store.Default 后可用；
// 对应 Lua 模块级 UserConfig）。
func Default() *UserConfig { return New(store.Default()) }

// Mine 便捷读取矿山配置段（对应 Lua UserConfig.get("mine")）。
func Mine() (config.MineConfig, error) {
	var cfg config.MineConfig
	err := Default().Get("mine", &cfg)
	return cfg, err
}

// Square 便捷读取布谷鸟广场配置段（对应 Lua UserConfig.get("square")）。
func Square() (config.SquareConfig, error) {
	var cfg config.SquareConfig
	err := Default().Get("square", &cfg)
	return cfg, err
}

// SeasideMarket 便捷读取海滩交易所配置段（对应 Lua UserConfig.get("seasideMarket")）。
func SeasideMarket() (config.SeasideMarketConfig, error) {
	var cfg config.SeasideMarketConfig
	err := Default().Get("seasideMarket", &cfg)
	return cfg, err
}

// Arena 便捷读取王国竞技场配置段（对应 Lua UserConfig.get("arena")）。
func Arena() (config.ArenaConfig, error) {
	var cfg config.ArenaConfig
	err := Default().Get("arena", &cfg)
	return cfg, err
}

// Starlight 便捷读取梦幻繁星岛配置段（对应 Lua UserConfig.get("starlight")）。
func Starlight() (config.StarlightConfig, error) {
	var cfg config.StarlightConfig
	err := Default().Get("starlight", &cfg)
	return cfg, err
}

// Biscuit 便捷读取洗脆饼配置段（对应 Lua UserConfig.get("biscuit")）。
func Biscuit() (config.BiscuitConfig, error) {
	var cfg config.BiscuitConfig
	err := Default().Get("biscuit", &cfg)
	return cfg, err
}

// Load 返回全部配置段（默认值 + 持久化覆盖合并后的副本）。
func (u *UserConfig) Load() (map[string]map[string]any, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cache == nil {
		if err := u.loadLocked(); err != nil {
			return nil, err
		}
	}
	out := make(map[string]map[string]any, len(u.cache))
	for name, section := range u.cache {
		out[name] = copySection(section)
	}
	return out, nil
}

func (u *UserConfig) loadLocked() error {
	saved, err := u.st.Get(storeKey, map[string]any{})
	if err != nil {
		return err
	}
	savedSections, ok := saved.(map[string]any)
	if !ok {
		savedSections = map[string]any{}
	}
	u.cache = make(map[string]map[string]any, len(defaultsMap))
	for name, def := range defaultsMap {
		section := copySection(def)
		if over, ok := savedSections[name].(map[string]any); ok {
			for k, v := range over {
				section[k] = v
			}
		}
		u.cache[name] = section
	}
	return nil
}

// Get 解码单个配置段到 out（默认值 + 保存覆盖合并后的结果）。
func (u *UserConfig) Get(section string, out any) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cache == nil {
		if err := u.loadLocked(); err != nil {
			return err
		}
	}
	merged, ok := u.cache[section]
	if !ok {
		return errUnknownSection
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// Set 合并部分字段到某段并落盘。
func (u *UserConfig) Set(section string, partial any) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cache == nil {
		if err := u.loadLocked(); err != nil {
			return err
		}
	}
	sectionMap, ok := u.cache[section]
	if !ok {
		return errUnknownSection
	}
	raw, err := json.Marshal(partial)
	if err != nil {
		return err
	}
	var patch map[string]any
	if err := json.Unmarshal(raw, &patch); err != nil {
		return err
	}
	for k, v := range patch {
		sectionMap[k] = v
	}
	return u.saveLocked()
}

func (u *UserConfig) saveLocked() error {
	sections := make(map[string]any, len(u.cache))
	for name, section := range u.cache {
		sections[name] = copySection(section)
	}
	return u.st.Set(storeKey, sections)
}

func copySection(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
