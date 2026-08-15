// Package square 对应 Lua 工程的 game/常规_布谷鸟广场/：广场特征库、页面、会话、路由与任务。
package square

import (
	"image"

	"app/internal/vision"
)

// HomeFeatures 广场首页特征（对应 Lua 广场_特征库.home）。
type HomeFeatures struct {
	Feature vision.Feature
	BackBtn image.Rectangle
}

// DialogLeaveFeatures 离开广场弹窗特征（对应 Lua 广场_特征库.dialogLeave）。
type DialogLeaveFeatures struct {
	Feature          vision.Feature
	CancelBtn        image.Rectangle
	LeaveBtn         image.Rectangle
	ConfirmRewardBtn image.Rectangle
	RewardNowOcr     image.Rectangle
	RewardTotalOcr   image.Rectangle
	DailyMaxOcr      image.Rectangle
	IsFinishOcr      image.Rectangle
}

// SquareFeatures 布谷鸟广场特征库（对应 Lua SquareFeatureLib）。
type SquareFeatures struct {
	Home        HomeFeatures
	DialogLeave DialogLeaveFeatures
}

var squareFeatures = SquareFeatures{
	Home: HomeFeatures{
		Feature: vision.Feature{Points: "1531|211|ffe314-101010,1533|68|36a3e3-101010,86|541|f9c16a-101010,59|298|fbe7ab-101010,59|96|ef345c-101010,67|110|95eb0e-101010", Sim: 0.9},
		BackBtn: image.Rect(1530, 39, 1543, 58),
	},
	DialogLeave: DialogLeaveFeatures{
		Feature:          vision.Feature{Points: "581|814|0ca5db-101010,407|819|87433b-101010,393|68|dd9387-101010,449|386|f5f3e3-101010,483|414|cd294e-101010,517|438|adf308-101010", Sim: 0.9},
		CancelBtn:        image.Rect(1214, 52, 1231, 70),
		LeaveBtn:         image.Rect(619, 790, 669, 817),
		ConfirmRewardBtn: image.Rect(930, 800, 983, 817),
		RewardNowOcr:     image.Rect(794, 342, 860, 374),
		RewardTotalOcr:   image.Rect(748, 382, 820, 414),
		DailyMaxOcr:      image.Rect(720, 406, 879, 483),
		IsFinishOcr:      image.Rect(754, 431, 848, 462),
	},
}

// FeatureLib 返回布谷鸟广场特征库（对应 SquareFeatureLib）。
func FeatureLib() SquareFeatures { return squareFeatures }
