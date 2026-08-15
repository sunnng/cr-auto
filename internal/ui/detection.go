// Local detection projection types. The diagnostic page renders whatever the
// host publishes as a frame observation; the UI package does not interpret
// these values, it only displays them.
package ui

type SceneID string

const SceneUnknown SceneID = "unknown"

type AnchorRole string

const (
	AnchorRequired  AnchorRole = "required"
	AnchorExclusion AnchorRole = "exclusion"
)

// Scene names kept in sync with the game contexts the UI can display. They are
// display keys, not a domain model; the host decides what each value means.
// The game-side scene scan keys (internal/game/detect.go) must match these.
const (
	SceneKingdomHome              SceneID = "kingdom_home"
	SceneKingdomEvent             SceneID = "kingdom_event"
	SceneKingdomAdventure         SceneID = "kingdom_adventure"
	SceneProductionAssistant      SceneID = "production_assistant"
	SceneProductionOverviewBase   SceneID = "production_overview"
	SceneProductionOverviewSparse SceneID = "production_overview_sparse"
	SceneProductionOverviewFilled SceneID = "production_overview_filled"
	SceneProductionFillDialog     SceneID = "production_fill_dialog"
	SceneLandmarkReward           SceneID = "landmark_reward"
	SceneUnstableNetwork          SceneID = "unstable_network"
	SceneMineHome                 SceneID = "mine_home"
	SceneMineVenture              SceneID = "mine_venture"
	SceneMineMining               SceneID = "mine_mining"
	SceneMineBattle               SceneID = "mine_battle"
	SceneMineJelly                SceneID = "mine_jelly"
	SceneSeasideMarket            SceneID = "seaside_market"
	SceneArenaLobby               SceneID = "arena_lobby"
	SceneStarlightHome            SceneID = "starlight_home"
	SceneSquareHome               SceneID = "square_home"
)

type Point struct {
	X int
	Y int
}

// Rect is an inclusive clickable rectangle in physical screen coordinates.
// Min is the top-left corner and Max is the bottom-right corner.
type Rect struct {
	Min Point
	Max Point
}

type SceneCandidate struct {
	Scene             SceneID
	Score             float32
	MatchedAnchors    int
	TotalAnchors      int
	RequiredMatched   int
	RequiredAnchors   int
	ExclusionMatches  int
	ExclusionAnchors  int
	ConstraintsPassed bool
}

type AnchorObservation struct {
	X        int
	Y        int
	Matched  bool
	Coverage float32
	Role     AnchorRole
}

type SlotObservation struct {
	X        int
	Y        int
	Occupied bool
}

type TargetObservation struct {
	ID         string
	Region     Rect
	Confidence float32
}

type TextObservation struct {
	Region     string
	Text       string
	Confidence float32
	X          int
	Y          int
	Width      int
	Height     int
}

type Detection struct {
	FrameID         uint64
	Scene           SceneID
	Confidence      float32
	Interrupt       bool
	Notice          bool
	Sensitive       bool
	RejectionReason string
	Candidates      []SceneCandidate
	Anchors         []AnchorObservation
	Slots           []SlotObservation
	OccupiedSlots   int
	TotalSlots      int
	SlotThreshold   int
	Targets         []TargetObservation
	Text            []TextObservation
	TextError       string
	SpendClass      string
}

// SceneDisplayName 场景键的中文显示名（识别诊断页用）。
// 未注册的场景键回退为原键（截断到 16 字符）。
func SceneDisplayName(scene string) string {
	switch SceneID(scene) {
	case SceneKingdomHome:
		return "王国主页"
	case SceneKingdomEvent:
		return "王国活动页"
	case SceneKingdomAdventure:
		return "王国探险页"
	case SceneProductionAssistant:
		return "生产助手"
	case SceneProductionOverviewBase:
		return "生产总览"
	case SceneProductionOverviewSparse:
		return "生产总览 · 未填满"
	case SceneProductionOverviewFilled:
		return "生产总览 · 已填满"
	case SceneProductionFillDialog:
		return "一次填满确认"
	case SceneLandmarkReward:
		return "地标奖励"
	case SceneUnstableNetwork:
		return "网络联机状态不稳定"
	case SceneMineHome:
		return "矿山首页"
	case SceneMineVenture:
		return "矿山勘查域"
	case SceneMineMining:
		return "矿山开采页"
	case SceneMineBattle:
		return "矿山战斗页"
	case SceneMineJelly:
		return "解除洋菜冻页"
	case SceneSeasideMarket:
		return "海滩交易所"
	case SceneArenaLobby:
		return "王国竞技场大厅"
	case SceneStarlightHome:
		return "梦幻繁星岛"
	case SceneSquareHome:
		return "布谷鸟广场"
	case SceneUnknown, "":
		return "未知场景"
	default:
		return limitRunes(scene, 16)
	}
}

func limitRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func cloneDetection(src Detection) Detection {
	dst := src
	dst.Candidates = append([]SceneCandidate(nil), src.Candidates...)
	dst.Anchors = append([]AnchorObservation(nil), src.Anchors...)
	dst.Slots = append([]SlotObservation(nil), src.Slots...)
	dst.Targets = append([]TargetObservation(nil), src.Targets...)
	dst.Text = append([]TextObservation(nil), src.Text...)
	return dst
}
