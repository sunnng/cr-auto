// Package popup 对应 Lua 工程的 game/通用_弹窗/：通用弹窗页面特征库。
package popup

import (
	"image"

	"app/internal/vision"
)

// UnstableNetwork 网络联机状态不稳定弹窗（对应 Lua page.lua 网络联机状态不稳定）。
type UnstableNetwork struct {
	Feature    vision.Feature
	ConfirmBtn image.Rectangle
}

// UnstableNetworkDef 返回“网络联机状态不稳定”弹窗定义（守卫注册用）。
func UnstableNetworkDef() UnstableNetwork {
	return UnstableNetwork{
		Feature:    vision.Feature{Points: "468|224|6a719f-101010,1132|235|363d5f-101010,789|415|505050-101010,790|462|505050-101010,476|672|dbcfc6-101010,1140|673|aea09b-101010,855|634|7ace0e-101010,795|631|ffffff-101010,695|613|95d83e-101010", Sim: 0.9},
		ConfirmBtn: image.Rect(775, 621, 828, 647),
	}
}
