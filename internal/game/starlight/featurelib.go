// Package starlight 对应 Lua 工程的 game/常规_梦幻繁星岛/：繁星岛坐标库、页面、会话、路由与任务。
package starlight

import (
	"image"

	"app/internal/vision"
)

// HomeFeatures 繁星岛首页特征（对应 Lua 繁星岛_坐标库.home）。
type HomeFeatures struct {
	Feature          vision.Feature
	SailingManualBtn image.Rectangle
	BackBtn          image.Rectangle
	TaskBtn          image.Rectangle
}

// TaskPageFeatures 任务页特征（对应 Lua 坐标库.任务）。
type TaskPageFeatures struct {
	Feature      vision.Feature
	ClaimableBtn vision.FindDef
	BackBtn      image.Rectangle
}

// ManualPageFeatures 航海手册页特征（对应 Lua 坐标库.航海手册）。
type ManualPageFeatures struct {
	Feature        vision.Feature
	LoginIslandBtn image.Rectangle
}

// VanillaPageFeatures 纯香草小岛页特征（对应 Lua 坐标库.纯香草小岛）。
type VanillaPageFeatures struct {
	Feature vision.Feature
	BackBtn image.Rectangle
}

// StarlightFeatures 梦幻繁星岛坐标库（对应 Lua StarlightFeatureLib）。
type StarlightFeatures struct {
	Home    HomeFeatures
	Task    TaskPageFeatures
	Manual  ManualPageFeatures
	Vanilla VanillaPageFeatures
}

var starlightFeatures = StarlightFeatures{
	Home: HomeFeatures{
		Feature:          vision.Feature{Points: "265|777|1a953c-101010,72|246|fbe7ab-101010,573|63|5cebaf-101010,1526|55|36a1e3-101010,1516|204|263e7a-101010,1462|801|b12d38-101010", Sim: 0.9},
		SailingManualBtn: image.Rect(1431, 800, 1453, 822),
		BackBtn:          image.Rect(1536, 51, 1548, 58),
		TaskBtn:          image.Rect(1526, 188, 1538, 201),
	},
	Task: TaskPageFeatures{
		Feature: vision.Feature{Points: "342|68|95979f-101010,1421|73|34a1e3-101010,886|81|ffd960-101010,104|796|4f0411-101010,1463|802|58161b-101010", Sim: 0.9},
		ClaimableBtn: vision.FindDef{
			Region:       image.Rect(216, 171, 1364, 618),
			FirstColor:   "e12d52-101010",
			OffsetColors: "-48|-39|93db04-101010|48|34|00a100-101010|16|-36|f7f3df-101010|-24|27|f1c1ab-101010|24|0|fd6e8b-101010|-18|-26|560308-101010",
			Dir:          0,
			Sim:          0.9,
		},
		BackBtn: image.Rect(1392, 67, 1407, 82),
	},
	Manual: ManualPageFeatures{
		Feature:        vision.Feature{Points: "67|83|dba12c-101010,1532|68|36a5e6-101010,367|775|7ace0e-101010,81|824|288ead-101010,1454|831|2d91af-101010", Sim: 0.9},
		LoginIslandBtn: image.Rect(428, 761, 464, 770),
	},
	Vanilla: VanillaPageFeatures{
		Feature: vision.Feature{Points: "29|36|ffffff-101010,463|55|ffffff-101010,1532|66|36a3e3-101010,1513|294|fff3a7-101010,1463|779|28a946-101010", Sim: 0.9},
		BackBtn: image.Rect(1536, 51, 1548, 58),
	},
}

// FeatureLib 返回梦幻繁星岛坐标库（对应 StarlightFeatureLib）。
func FeatureLib() StarlightFeatures { return starlightFeatures }
