// Package arena 对应 Lua 工程的 game/常规_王国竞技场/：竞技场特征库、页面、会话、路由与任务。
package arena

import (
	"image"

	"app/internal/vision"
)

// LobbyFeatures 竞技场大厅特征（对应 Lua 竞技场_特征库.lobby）。
type LobbyFeatures struct {
	Feature          vision.Feature
	CloseBtn         image.Rectangle
	MedalTicketOcr   image.Rectangle
	TrophyOcr        image.Rectangle
	RefreshOcr       image.Rectangle
	FreeRefreshOcr   image.Rectangle
	FreeRefreshTap   image.Point
	BuyTicketBtn     image.Point
	BuyTicketSlider  image.Rectangle
	BuyTicketConfirm image.Point
}

// OpponentFeatures 对手扫描特征（对应 Lua 竞技场_特征库.opponent）。
// 注意：Lua 的 opponent.numberOcr 含引擎调参（detScaleRatio/binaryThresh 等），
// 因 OCR 引擎走注入接缝（ADR-0003）无法逐调用传入，由设备端引擎统一配置。
type OpponentFeatures struct {
	FindDef      vision.FindDef
	BaseSite     image.Point
	TrophyRect   image.Rectangle
	ResultOffset image.Point
	ResultColors struct {
		Win  string
		Draw string
		Lose string
	}
}

// TeamSelectFeatures 队伍选择页特征（对应 Lua 竞技场_特征库.teamSelect）。
type TeamSelectFeatures struct {
	Feature     vision.Feature
	StartBattle image.Point
}

// ArenaDialogFeatures 竞技场弹窗特征（对应 Lua 竞技场_特征库.dialog）。
type ArenaDialogFeatures struct {
	MissingTopping struct {
		Feature vision.Feature
		Confirm image.Point
	}
	DeployMore struct {
		Feature vision.Feature
		Confirm image.Point
	}
}

// SettlementFeatures 结算页特征（对应 Lua 竞技场_特征库.settlement）。
type SettlementFeatures struct {
	Feature      vision.Feature
	ResultOcr    image.Rectangle
	LeaveFeature vision.Feature
	LeaveBtn     image.Rectangle
}

// PaginationFeatures 分页滑动参数（对应 Lua 竞技场_特征库.pagination.swipeLeft）。
type PaginationFeatures struct {
	SwipeLeft struct{ X1, Y1, X2, Y2 int }
}

// ArenaFeatures 王国竞技场特征库（对应 Lua ArenaFeatureLib）。
type ArenaFeatures struct {
	Lobby      LobbyFeatures
	Opponent   OpponentFeatures
	TeamSelect TeamSelectFeatures
	Dialog     ArenaDialogFeatures
	Settlement SettlementFeatures
	Pagination PaginationFeatures
}

func findDef(x1, y1, x2, y2 int, first, offsets string, dir int, sim float32) vision.FindDef {
	return vision.FindDef{
		Region:       image.Rect(x1, y1, x2, y2),
		FirstColor:   first,
		OffsetColors: offsets,
		Dir:          dir,
		Sim:          sim,
	}
}

var arenaFeatures = ArenaFeatures{
	Lobby: LobbyFeatures{
		Feature:          vision.Feature{Points: "74|186|33f2f8-101010,75|173|ffe400-101010,74|313|cf7b34-101010,75|512|be6928-101010,72|389|ef8421-101010", Sim: 0.95},
		CloseBtn:         image.Rect(1530, 14, 1584, 77),
		MedalTicketOcr:   image.Rect(876, 20, 1270, 77),
		TrophyOcr:        image.Rect(177, 733, 359, 777),
		RefreshOcr:       image.Rect(1345, 733, 1549, 777),
		FreeRefreshOcr:   image.Rect(1332, 735, 1558, 773),
		FreeRefreshTap:   image.Point{X: 1421, Y: 758},
		BuyTicketBtn:     image.Point{X: 1213, Y: 48},
		BuyTicketSlider:  image.Rect(605, 635, 731, 635),
		BuyTicketConfirm: image.Point{X: 1056, Y: 609},
	},
	Opponent: OpponentFeatures{
		FindDef:      findDef(560, 478, 1589, 556, "1cc2e2-101010", "-1|7|fbcf00-101010|-1|-12|c65d00-101010|-11|-2|e32840-101010|1|18|fedd00-101010", 0, 0.95),
		BaseSite:     image.Point{X: 643, Y: 531},
		TrophyRect:   image.Rect(664, 521, 806, 556),
		ResultOffset: image.Point{X: 83, Y: -109},
		ResultColors: struct {
			Win  string
			Draw string
			Lose string
		}{Win: "ccff33", Draw: "66ffff", Lose: "ff9999"},
	},
	TeamSelect: TeamSelectFeatures{
		Feature:     vision.Feature{Points: "424|840|7ace0e-101010,42|835|3db8e5-101010,1477|794|7ace0e-101010,1264|825|e5b129-101010", Sim: 0.95},
		StartBattle: image.Point{X: 1408, Y: 823},
	},
	Dialog: ArenaDialogFeatures{
		MissingTopping: struct {
			Feature vision.Feature
			Confirm image.Point
		}{
			Feature: vision.Feature{Points: "620|682|0ca6df-101010,1008|680|7ace0e-101010,807|195|363d5f-101010,809|804|ffffff-101010", Sim: 0.95},
			Confirm: image.Point{X: 946, Y: 678},
		},
		DeployMore: struct {
			Feature vision.Feature
			Confirm image.Point
		}{
			Feature: vision.Feature{Points: "908|632|7ace0e-101010,695|632|0ca6df-101010,824|246|363d5f-101010", Sim: 0.95},
			Confirm: image.Point{X: 960, Y: 636},
		},
	},
	Settlement: SettlementFeatures{
		Feature:      vision.Feature{Points: "1454|46|333333-101010,1451|49|ffffff-101010,1442|49|6786bd-101010,1467|56|1b2850-101010,835|94|ffff66-101010", Sim: 0.9},
		ResultOcr:    image.Rect(738, 151, 870, 203),
		LeaveFeature: vision.Feature{Points: "1523|811|0ca6df-101010,1158|809|f67b4b-101010,1532|38|34a0e4-101010,1149|781|ffffff-101010", Sim: 0.9},
		LeaveBtn:     image.Rect(1424, 785, 1457, 799),
	},
	Pagination: PaginationFeatures{
		SwipeLeft: struct{ X1, Y1, X2, Y2 int }{X1: 1524, Y1: 534, X2: 877, Y2: 533},
	},
}

// FeatureLib 返回王国竞技场特征库（对应 ArenaFeatureLib）。
func FeatureLib() ArenaFeatures { return arenaFeatures }
