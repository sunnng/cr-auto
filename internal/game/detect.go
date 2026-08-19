// Package game 对应 Lua 工程的 game/ 目录：任务构建器与业务注册。
package game

import (
	"sort"

	"app/internal/game/arena"
	"app/internal/game/kingdom"
	"app/internal/game/mine"
	"app/internal/game/popup"
	"app/internal/game/seaside"
	"app/internal/game/square"
	"app/internal/game/starlight"
	"app/internal/lib/color"
	"app/internal/vision"
)

// 识别诊断页的场景键。键值必须与 internal/ui 的 SceneID 常量保持一致
// （ui 层负责显示名映射；见 ui/detection.go 顶部注释）。
const (
	SceneUnstableNetwork  = "unstable_network"
	SceneKingdomHome      = "kingdom_home"
	SceneKingdomEvent     = "kingdom_event"
	SceneKingdomAdventure = "kingdom_adventure"
	SceneMineHome         = "mine_home"
	SceneMineVenture      = "mine_venture"
	SceneMineMining       = "mine_mining"
	SceneMineBattle       = "mine_battle"
	SceneMineJelly        = "mine_jelly"
	SceneSeasideMarket    = "seaside_market"
	SceneArenaLobby       = "arena_lobby"
	SceneStarlightHome    = "starlight_home"
	SceneSquareHome       = "square_home"
)

// sceneDef 识别诊断页的一个场景：展示键 + 一到多个可选特征（任一命中即视为该场景）。
type sceneDef struct {
	Key      string
	Features []vision.Feature
}

// sceneRegistry 识别诊断页的场景集合（“当前在哪个页面”的比色证据）。
// 通用弹窗优先，其余按模块注册顺序。
var sceneRegistry = []sceneDef{
	{Key: SceneUnstableNetwork, Features: []vision.Feature{popup.UnstableNetworkDef().Feature}},
	{Key: SceneKingdomHome, Features: []vision.Feature{kingdom.Home().Feature}},
	{Key: SceneKingdomEvent, Features: []vision.Feature{kingdom.Event().Feature}},
	{Key: SceneKingdomAdventure, Features: []vision.Feature{kingdom.Adventure().Feature}},
	{Key: SceneMineHome, Features: []vision.Feature{mine.MineHome().Feature}},
	{Key: SceneMineVenture, Features: []vision.Feature{
		mine.MineVenture().Setup.Feature,
		mine.MineVenture().Ready.Feature,
		mine.MineVenture().Running.Feature,
		mine.MineVenture().Settle.Feature,
	}},
	{Key: SceneMineMining, Features: []vision.Feature{mine.Mining().Page.Feature}},
	{Key: SceneMineBattle, Features: []vision.Feature{mine.Battle().Feature}},
	{Key: SceneMineJelly, Features: []vision.Feature{mine.Jelly().Feature}},
	{Key: SceneSeasideMarket, Features: []vision.Feature{seaside.FeatureLib().Page.Feature}},
	{Key: SceneArenaLobby, Features: []vision.Feature{arena.FeatureLib().Lobby.Feature}},
	{Key: SceneStarlightHome, Features: []vision.Feature{starlight.FeatureLib().Home.Feature}},
	{Key: SceneSquareHome, Features: []vision.Feature{square.FeatureLib().Home.Feature}},
}

// SceneKeys 返回识别诊断页注册的全部场景键（与 ui.SceneID 同步，见 catalog_test）。
func SceneKeys() []string {
	keys := make([]string, 0, len(sceneRegistry))
	for _, scene := range sceneRegistry {
		keys = append(keys, scene.Key)
	}
	return keys
}

// SceneCandidate 识别诊断页的单个场景候选：命中色点数与置信度。
type SceneCandidate struct {
	Key     string
	Matched int
	Total   int
	Score   float32
}

// SceneDetection 识别诊断页的整帧扫描结果。
type SceneDetection struct {
	Best       string
	Confidence float32
	Candidates []SceneCandidate
	// Anchors 最佳场景命中特征的逐点结果（诊断页锚点叠加用）。
	Anchors []vision.PointResult
}

// DetectScene 扫描全部注册场景，返回置信度最高的场景与全部候选（按置信度降序）。
// 无任何场景命中时 Best 为空串、Anchors 为空。
func DetectScene() SceneDetection {
	var detection SceneDetection
	color.Session(func() {
		var best sceneDef
		var bestPoints []vision.PointResult
		bestScore := float32(0)
		candidates := make([]SceneCandidate, 0, len(sceneRegistry))

		for _, scene := range sceneRegistry {
			matched, total, score, points := sceneScore(scene)
			if total == 0 || score <= 0 {
				continue
			}
			candidates = append(candidates, SceneCandidate{
				Key:     scene.Key,
				Matched: matched,
				Total:   total,
				Score:   score,
			})
			if score > bestScore {
				bestScore = score
				best = scene
				bestPoints = points
			}
		}

		sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })

		if len(candidates) == 0 {
			return
		}
		detection = SceneDetection{
			Best:       best.Key,
			Confidence: bestScore,
			Candidates: candidates,
			Anchors:    bestPoints,
		}
	})
	return detection
}

// sceneScore 计算单个场景的（命中数, 总点数, 置信度, 逐点结果）；
// 场景含多个特征时取置信度最高的特征。置信度 = 命中数 / 总点数。
func sceneScore(scene sceneDef) (matched, total int, score float32, points []vision.PointResult) {
	for _, feature := range scene.Features {
		results, _ := color.MatchPoints(feature)
		if len(results) == 0 {
			continue
		}
		hit := 0
		for _, r := range results {
			if r.Matched {
				hit++
			}
		}
		ratio := float32(hit) / float32(len(results))
		if ratio > score {
			score = ratio
			matched = hit
			total = len(results)
			points = results
		}
	}
	return matched, total, score, points
}
