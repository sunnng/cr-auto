// Package kingdom 对应 Lua 工程的 game/通用_王国/：王国首页/活动/探险页特征库与页面。
package kingdom

import (
	"image"

	"app/internal/vision"
)

// HomeFeatures 王国首页特征（对应 Lua 特征库.kingdomHome）。
type HomeFeatures struct {
	Feature      vision.Feature
	EventBtn     image.Rectangle
	SquareBtn    image.Rectangle
	AdventureBtn image.Rectangle
}

// EventFeatures 王国活动页特征（对应 Lua 特征库.kingdomEvent）。
type EventFeatures struct {
	Feature          vision.Feature
	MineBtn          image.Rectangle
	SeasideMarketBtn image.Rectangle
	StarlightBtn     image.Rectangle
}

// AdventureFeatures 王国探险页特征（对应 Lua 特征库.kingdomAdventure）。
type AdventureFeatures struct {
	Feature  vision.Feature
	ArenaOcr image.Rectangle
}

var home = HomeFeatures{
	Feature:      vision.Feature{Points: "1380|60|f7e5cb-101010,59|323|b3001b-101010,96|825|fbed78-101010,274|838|85b7f7-101010,1535|854|2f4c6c-101010,1311|845|d99f26-101010", Sim: 0.9},
	EventBtn:     image.Rect(256, 800, 269, 831),
	SquareBtn:    image.Rect(589, 811, 616, 830),
	AdventureBtn: image.Rect(1371, 812, 1403, 831),
}

var event = EventFeatures{
	Feature:          vision.Feature{Points: "1311|843|261d06-101010,722|820|2d1f00-101010,235|825|2d2718-101010,69|805|252625-101010,71|332|2e2a1f-101010,795|140|b59756-101010,1290|65|2a1c0f-101010,1551|69|36a3e3-101010", Sim: 0.9},
	MineBtn:          image.Rect(1228, 578, 1253, 601),
	SeasideMarketBtn: image.Rect(574, 582, 593, 604),
	StarlightBtn:     image.Rect(1225, 366, 1252, 387),
}

var adventure = AdventureFeatures{
	Feature:  vision.Feature{Points: "40|61|61a1eb-101010,59|50|dde7e7-101010,28|72|f3cf4e-101010,71|73|d7ad37-101010,1552|70|36a5e3-101010,1066|65|07b3fb-101010,782|62|eba900-101010", Sim: 0.9},
	ArenaOcr: image.Rect(39, 201, 1592, 282),
}

// Home 返回王国首页特征（对应 KingdomFeatureLib.home()）。
func Home() HomeFeatures { return home }

// Event 返回王国活动页特征（对应 KingdomFeatureLib.event()）。
func Event() EventFeatures { return event }

// Adventure 返回王国探险页特征（对应 KingdomFeatureLib.adventure()）。
func Adventure() AdventureFeatures { return adventure }
