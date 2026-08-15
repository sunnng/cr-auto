// Package seaside 对应 Lua 工程的 game/常规_海滩交易所/：交易所坐标库、页面、会话、路由与任务。
package seaside

import (
	"image"

	"app/internal/vision"
)

// PageFeatures 交易所页特征（对应 Lua 交易所_坐标库.page）。
type PageFeatures struct {
	Feature          vision.Feature
	RefreshBtn       image.Rectangle
	RefreshStatusOcr image.Rectangle
	CanRefreshOcr    image.Rectangle
	CloseBtn         image.Rectangle
	RefreshOcr       image.Rectangle
}

// DialogConfirmFeatures 购买确认弹窗特征（对应 Lua 坐标库.dialogConfirm）。
type DialogConfirmFeatures struct {
	Feature    vision.Feature
	ConfirmBtn image.Rectangle
	CancelBtn  image.Rectangle
}

// ItemShortageFeatures 道具不足弹窗特征（对应 Lua 坐标库.itemShortageDialog）。
type ItemShortageFeatures struct {
	Feature         vision.Feature
	ItemShortageOcr image.Rectangle
	CancelBtn       image.Rectangle
}

// TabFeatures 标签页特征（对应 Lua 坐标库.tab）。
type TabFeatures struct {
	ItemExchangeTab struct {
		Area image.Rectangle
	}
}

// ListFeatures 商品列表翻页特征（对应 Lua 坐标库.list）。
type ListFeatures struct {
	ArrowRight vision.FindDef
	Swipe      touchSwipe
	MaxSwipes  int
}

// SlotFeatures 货架槽位几何参数（对应 Lua 坐标库.slot）。
type SlotFeatures struct {
	BuyBtnOffsetY int
	BuyBtnHalfW   int
	BuyBtnHalfH   int
	CrateHalfW    int
	CrateHalfH    int
	CrateOffsetY  int
}

// SeasideFeatures 海滩交易所坐标库（对应 Lua SeasideMarketFeatureLib）。
type SeasideFeatures struct {
	Page     PageFeatures
	Dialog   DialogConfirmFeatures
	Shortage ItemShortageFeatures
	Tab      TabFeatures
	List     ListFeatures
	Slot     SlotFeatures
	Stock    map[string]vision.FindDef
}

// touchSwipe 滑动参数（对应 Lua 坐标库 list.swipe）。
type touchSwipe struct {
	X1, Y1, X2, Y2 int
	HoldMs         int
	UpMs           int
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

// StockDef 商品找色定义（对应 Lua Stock，键名为配置 items 中文名）。
type StockDef map[string]vision.FindDef

var seasideFeatures = func() SeasideFeatures {
	var f SeasideFeatures
	f.Page = PageFeatures{
		Feature:          vision.Feature{Points: "37|723|f51b67-101010,189|95|14633c-101010,477|128|b2d155-101010,448|285|f5365e-101010,1409|828|895b3d-101010,1508|320|57432a-101010", Sim: 0.9},
		RefreshBtn:       image.Rect(1419, 458, 1459, 480),
		RefreshStatusOcr: image.Rect(1298, 447, 1557, 488),
		CanRefreshOcr:    image.Rect(1358, 449, 1496, 486),
		CloseBtn:         image.Rect(1530, 14, 1584, 77),
		RefreshOcr:       image.Rect(1298, 447, 1557, 488),
	}
	f.Dialog = DialogConfirmFeatures{
		Feature:    vision.Feature{Points: "120|127|126345-101010,38|720|7f0e37-101010,339|241|2e5825-101010,478|128|5a692a-101010,1152|237|36a6e8-101010,455|220|686f9d-101010,1471|829|44351e-101010", Sim: 0.9},
		ConfirmBtn: image.Rect(776, 621, 829, 646),
		CancelBtn:  image.Rect(1143, 211, 1159, 229),
	}
	f.Shortage = ItemShortageFeatures{
		Feature:         vision.Feature{Points: "1559|854|261010-101010,27|717|710b2d-101010,131|222|1a3722-101010,351|273|7f7f55-101010,475|129|58682a-101010,517|246|68709d-101010,1092|265|36a5e5-101010", Sim: 0.9},
		ItemShortageOcr: image.Rect(715, 331, 885, 376),
		CancelBtn:       image.Rect(1086, 231, 1102, 253),
	}
	f.Tab.ItemExchangeTab.Area = image.Rect(559, 831, 643, 862)
	f.List = ListFeatures{
		ArrowRight: findDef(1524, 616, 1577, 684, "000000-101010|000000-101010|000000-101010|030303-101010|030303-101010|12110d-101010", "", 0, 0.9),
		Swipe:      touchSwipe{X1: 1500, Y1: 650, X2: 100, Y2: 650, HoldMs: 1200, UpMs: 1200},
		MaxSwipes:  20,
	}
	f.Slot = SlotFeatures{
		BuyBtnOffsetY: 110,
		BuyBtnHalfW:   105,
		BuyBtnHalfH:   24,
		CrateHalfW:    90,
		CrateHalfH:    65,
		CrateOffsetY:  -20,
	}
	f.Stock = StockDef{
		"灿烂的光之碎片": findDef(3, 602, 1587, 707, "a9e4ff-101010",
			"-7|-1|fffff3-101010|-34|-1|cfefb9-101010|14|1|cf97f6-101010|8|20|ef84a9-101010|-3|27|fad4a2-101010|-14|29|ee7fe3-101010|38|16|320f5d-101010|-24|-19|31105a-101010", 0, 0.9),
		"十分钟加速券": findDef(3, 602, 1587, 707, "ffffff-101010",
			"0|-1|ffffff-101010|-4|12|2bd0e9-101010|-1|-25|6786bd-101010|-19|24|4168ad-101010|-40|0|6496c9-101010|37|-12|f9ffff-101010|36|17|c8e9f6-101010", 0, 0.9),
		"商品1_金紫": findDef(3, 602, 1587, 707, "b5a7f9-101010",
			"-65|10|80824f-101010|5|5|ffffff-101010|0|10|ada2fa-101010|10|-5|edcffc-101010", 0, 0.9),
		"商品2_蓝盒": findDef(3, 602, 1587, 707, "99bdff-101010",
			"0|-5|403f5e-101010|-15|0|2b2183-101010|5|0|201d53-101010|0|20|222255-101010", 0, 0.9),
		"商品3_罗盘": findDef(3, 602, 1587, 707, "c6c0fa-101010",
			"-10|-10|8974e8-101010|10|-10|cec0f8-101010|0|10|fefefe-101010|15|0|ffffff-101010", 0, 0.9),
		"商品4_绿书": findDef(3, 602, 1587, 707, "827d45-101010",
			"-20|0|837e47-101010|70|0|6b4eff-101010|80|0|6b4eff-101010|0|-20|837c46-101010|0|20|857c41-101010", 0, 0.9),
		"商品5_卷轴": findDef(3, 602, 1587, 707, "896dfc-101010",
			"-10|-10|fb1efe-101010|10|-10|ffffff-101010|0|10|c5b5ff-101010|15|0|ffffff-101010", 0, 0.9),
	}
	return f
}()

// FeatureLib 返回海滩交易所坐标库（对应 SeasideMarketFeatureLib）。
func FeatureLib() SeasideFeatures { return seasideFeatures }
