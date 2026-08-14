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
const (
	SceneKingdomHome              SceneID = "kingdom_home"
	SceneProductionAssistant      SceneID = "production_assistant"
	SceneProductionOverviewBase   SceneID = "production_overview"
	SceneProductionOverviewSparse SceneID = "production_overview_sparse"
	SceneProductionOverviewFilled SceneID = "production_overview_filled"
	SceneProductionFillDialog     SceneID = "production_fill_dialog"
	SceneLandmarkReward           SceneID = "landmark_reward"
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

func cloneDetection(src Detection) Detection {
	dst := src
	dst.Candidates = append([]SceneCandidate(nil), src.Candidates...)
	dst.Anchors = append([]AnchorObservation(nil), src.Anchors...)
	dst.Slots = append([]SlotObservation(nil), src.Slots...)
	dst.Targets = append([]TargetObservation(nil), src.Targets...)
	dst.Text = append([]TextObservation(nil), src.Text...)
	return dst
}
