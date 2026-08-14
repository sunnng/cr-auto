// Package mine 对应 Lua 工程的 game/常规_未知的地底矿山/：矿山特征库与矿山首页页面。
package mine

import (
	"image"

	"app/internal/vision"
)

// FindDef 找色定义（findMultiColorT 参数：区域、首色、相对色点、方向、相似度）。
// OffsetColors 沿用 Lua 特征库的管道分隔三元组格式。
func findDef(x1, y1, x2, y2 int, first, offsets string, dir int, sim float32) vision.FindDef {
	return vision.FindDef{
		Region:       image.Rect(x1, y1, x2, y2),
		FirstColor:   first,
		OffsetColors: offsets,
		Dir:          dir,
		Sim:          sim,
	}
}

var commonBackBtn = image.Rect(1546, 39, 1559, 53)

// MineHomeFeatures 矿山首页特征（对应 Lua mineHome）。
type MineHomeFeatures struct {
	Feature                       vision.Feature
	HasMiningCompletedTaskFeature vision.Feature
	VentureBtn                    image.Rectangle
	MiningBtn                     image.Rectangle
	BattleBtn                     image.Rectangle
	BackBtn                       image.Rectangle
}

// MineVentureSetup 勘查准备页特征。
type MineVentureSetup struct {
	Feature       vision.Feature
	StartBtn      image.Rectangle
	AutoSelectBtn image.Rectangle
}

// MineVentureReady 勘查就绪页特征。
type MineVentureReady struct {
	Feature  vision.Feature
	StartBtn image.Rectangle
}

// MineVentureRunning 勘查运行中页特征。
type MineVentureRunning struct {
	Feature  vision.Feature
	StopBtn  image.Rectangle
	FloorOcr image.Rectangle
}

// MineVentureSettle 勘查结算页特征。
type MineVentureSettle struct {
	Feature   vision.Feature
	FinishBtn image.Rectangle
}

// MineVentureDialog 勘查弹窗特征。
type MineVentureDialog struct {
	Feature    vision.Feature
	ConfirmBtn image.Rectangle
}

// MineVentureFeatures 勘查域特征（对应 Lua mineVenture）。
type MineVentureFeatures struct {
	JellyBtn            image.Rectangle
	Setup               MineVentureSetup
	Ready               MineVentureReady
	Running             MineVentureRunning
	Settle              MineVentureSettle
	DialogInfo          MineVentureDialog
	DialogConfirmCookie MineVentureDialog
	DialogStop          struct {
		Feature        vision.Feature
		ConfirmStopBtn image.Rectangle
	}
	BackBtn image.Rectangle
}

var mineHome = MineHomeFeatures{
	Feature:                       vision.Feature{Points: "1531|49|34a1e3,1379|60|f7e5cb,1291|58|e79d56,57|235|b3001c,73|256|f7e1b3,66|655|df994c,282|796|64677f,97|804|f37f0a,432|761|832427,1277|773|fffdf3,1298|791|db71c3", Sim: 0.9},
	HasMiningCompletedTaskFeature: vision.Feature{Points: "1565|747|ffffff-101010,1570|751|ff0000-101010,1558|757|000000-101010,1483|790|2e9ddf-101010,1535|767|ef6693-101010,1511|758|ffebff-101010,1454|840|efa150-101010", Sim: 0.9},
	VentureBtn:                    image.Rect(1271, 799, 1307, 831),
	MiningBtn:                     image.Rect(1457, 791, 1499, 822),
	BattleBtn:                     image.Rect(1505, 651, 1528, 675),
	BackBtn:                       commonBackBtn,
}

var mineVenture = func() MineVentureFeatures {
	var f MineVentureFeatures
	f.JellyBtn = image.Rect(419, 788, 435, 808)
	f.Setup = MineVentureSetup{
		Feature:       vision.Feature{Points: "79|178|fff49f-101010,85|184|4a496b-101010,401|660|08a6de-101010,491|667|ffd300-101010,558|747|a56142-101010,1184|304|8b4000-101010,1196|291|ce7931-101010,1322|833|a1a1a1-101010,1315|839|ffffff-101010", Sim: 0.9},
		StartBtn:      image.Rect(1217, 815, 1301, 846),
		AutoSelectBtn: image.Rect(545, 643, 630, 669),
	}
	f.Ready = MineVentureReady{
		Feature:  vision.Feature{Points: "1337|837|7ace0e-101010,657|657|ffd200-101010,398|659|0ca6df-101010,682|751|a8623f-101010,80|179|fbe788-101010,1217|292|cc4f55-101010", Sim: 0.95},
		StartBtn: image.Rect(1217, 815, 1301, 846),
	}
	f.Running = MineVentureRunning{
		Feature:  vision.Feature{Points: "78|177|fdf7a1-101010,219|51|ffffff-101010,573|833|0ca6df-101010,1191|292|d38235-101010,1290|684|d4b89a-101010", Sim: 0.95},
		StopBtn:  image.Rect(379, 816, 515, 850),
		FloorOcr: image.Rect(222, 138, 686, 203),
	}
	f.Settle = MineVentureSettle{
		Feature:   vision.Feature{Points: "233|612|d88c16-101010,1576|795|da9019-101010,913|185|ff2323-101010,903|208|9d9d9d-101010,901|196|deac51-101010", Sim: 0.95},
		FinishBtn: image.Rect(708, 815, 888, 860),
	}
	f.DialogInfo = MineVentureDialog{
		Feature:    vision.Feature{Points: "898|617|7bcf10-101010,864|263|ffffff-101010,867|264|020101-101010,704|368|efebe7-101010,710|368|4269ad-101010,1110|264|39a6e7-101010", Sim: 0.9},
		ConfirmBtn: image.Rect(760, 604, 849, 629),
	}
	f.DialogConfirmCookie = MineVentureDialog{
		Feature:    vision.Feature{Points: "1005|630|7bcf10-101010,730|630|08a6de-101010,830|248|393c63-101010,890|420|505050-101010,893|419|f7ebde-101010,919|506|8c8c8c-101010", Sim: 0.9},
		ConfirmBtn: image.Rect(916, 618, 977, 645),
	}
	f.DialogStop.Feature = vision.Feature{Points: "1062|629|f45a1e-101010,745|626|0ca6df-101010,892|246|363d5f-101010,75|197|553a23-101010,556|833|021821-101010", Sim: 0.95}
	f.DialogStop.ConfirmStopBtn = image.Rect(905, 608, 1004, 650)
	f.BackBtn = commonBackBtn
	return f
}()

// MiningRewardPage 开采奖励页特征。
type MiningRewardPage struct {
	TitleText  string
	TitleOcr   image.Rectangle
	ConfirmBtn image.Rectangle
}

// MiningCardSelect 选卡页特征。
type MiningCardSelect struct {
	ConfirmBtn   image.Rectangle
	BackBtn      image.Rectangle
	SelectedMark vision.FindDef // 已选矿卡角标/勾选；零值表示未配置
}

// MiningDialog 开采弹窗特征。
type MiningDialog struct {
	Feature          vision.Feature
	ConfirmBtn       image.Rectangle
	TodayNotAskAgain image.Rectangle
}

// MiningFeatures 开采页特征（对应 Lua mining）。
type MiningFeatures struct {
	NoMineCardOcr       image.Rectangle
	FreeLocationFeature vision.FindDef
	FreePlusFeature     vision.FindDef
	CanChooseNum        image.Rectangle
	MultiSelectBtn      image.Rectangle
	MultiSelectOcr      image.Rectangle
	CardListStartOcr    image.Rectangle
	CardListEndOcr      image.Rectangle
	Page                struct {
		Feature vision.Feature
	}
	CompletedTask vision.FindDef
	RewardPage    MiningRewardPage
	CardSelect    MiningCardSelect
	Home          struct {
		Feature vision.Feature // 未配置（零值）
	}
	StartableCard            vision.FindDef
	SetupFeature             vision.Feature
	SetupReadyFeature        vision.Feature
	DialogConfirmCookie      MiningDialog
	DialogCookieCountWarning MiningDialog
	AutoSelectCookieBtn      image.Rectangle
	ConfirmStartBtn          image.Rectangle
	BackBtn                  image.Rectangle
}

var mining = func() MiningFeatures {
	var f MiningFeatures
	f.NoMineCardOcr = image.Rect(662, 503, 915, 542)
	f.FreeLocationFeature = findDef(27, 165, 1580, 359, "c67654-101010",
		"-4|-26|f5ece4-101010|103|46|2f1e1b-101010|47|106|2f1e1b-101010|-50|110|2f1e1b-101010|-95|-58|392520-101010|90|-57|392520-101010|-68|120|804a40-101010", 0, 0.9)
	f.FreePlusFeature = findDef(1320, 60, 1575, 220, "f5c079-101010",
		"-31|-12|f9bd76-101010|-3|-44|f7b873-101010|28|-10|f3aa69-101010|-1|33|f1a365-101010|51|-51|2e1d1d-101010|-60|54|2d1b1b-101010", 0, 0.9)
	f.CanChooseNum = image.Rect(769, 740, 841, 785)
	f.MultiSelectBtn = image.Rect(129, 828, 265, 875)
	f.MultiSelectOcr = image.Rect(129, 828, 265, 875)
	f.CardListStartOcr = image.Rect(95, 452, 390, 532)
	f.CardListEndOcr = image.Rect(1210, 452, 1505, 532)
	f.Page.Feature = vision.Feature{Points: "219|48|ffffff-101010,176|57|ffffff-101010,903|65|a8623f-101010,1318|64|09c4ff-101010,1316|52|0589ff-101010,1547|68|36a6e6-101010,417|891|4b1d00-101010,1571|882|2c0d00-101010", Sim: 0.9}
	f.CompletedTask = findDef(34, 101, 1570, 251, "9bd400-101010",
		"-10|0|97d501-101010|-17|4|befd00-101010|-7|5|97d400-101010|-69|77|333333-101010|-76|68|d7e1f0-101010|-75|79|ffffff-101010|-63|87|4169ac-101010|-24|79|90f90a-101010|36|73|c9ff3a-101010", 0, 0.9)
	f.RewardPage = MiningRewardPage{
		TitleText:  "获得开采奖励",
		TitleOcr:   image.Rect(284, 204, 891, 312),
		ConfirmBtn: image.Rect(678, 762, 926, 799),
	}
	f.CardSelect = MiningCardSelect{
		ConfirmBtn: image.Rect(955, 742, 1015, 769),
		BackBtn:    image.Rect(1516, 14, 1584, 77),
		// SelectedMark 未配置（实机采图后填入）。
	}
	f.StartableCard = findDef(15, 93, 1588, 182, "ffffff-101010",
		"-11|-14|ffffff-101010|12|-15|fd7430-101010|12|15|ef1909-101010|1|20|ed1608-101010|-16|13|ef1808-101010|-17|15|230a0b-101010|14|-16|000000-101010", 0, 0.9)
	f.SetupFeature = vision.Feature{Points: "1487|827|a0a0a0-101010,1244|806|ffffff-101010,1277|570|ffffff-101010,1525|431|ffd200-101010,1516|499|0ca6df-101010", Sim: 0.9}
	f.SetupReadyFeature = vision.Feature{Points: "1411|824|ffffff-101010,1388|808|7ace0e-101010,1247|803|ffffff-101010", Sim: 0.9}
	f.DialogConfirmCookie = MiningDialog{
		Feature:          vision.Feature{Points: "468|223|6a719e-101010,1126|223|363d5f-101010,1131|683|afa09c-101010,474|664|dbcfc6-101010,571|638|0ca6df-101010,1069|644|7ace0e-101010,814|419|505050-101010,794|498|8c8c8c-101010,793|360|505050-101010", Sim: 0.9},
		ConfirmBtn:       image.Rect(932, 619, 972, 642),
		TodayNotAskAgain: image.Rect(871, 724, 887, 740),
	}
	f.DialogCookieCountWarning = MiningDialog{
		Feature:          vision.Feature{Points: "682|414|505050-101010,897|425|505050-101010,864|471|505050-101010,706|470|505050-101010,512|631|3db8e5-101010,1093|643|7ace0e-101010,1129|227|363d5f-101010,472|222|6a719e-101010", Sim: 0.9},
		ConfirmBtn:       image.Rect(930, 623, 959, 646),
		TodayNotAskAgain: image.Rect(878, 728, 886, 740),
	}
	f.AutoSelectCookieBtn = image.Rect(1424, 412, 1497, 432)
	f.ConfirmStartBtn = image.Rect(1363, 816, 1451, 835)
	f.BackBtn = image.Rect(1516, 14, 1584, 77)
	return f
}()

// BattleQuickDialog 快转弹窗特征。
type BattleQuickDialog struct {
	Feature    vision.Feature
	ConfirmBtn image.Rectangle
	CancelBtn  image.Rectangle
	CountOcr   image.Rectangle
}

// BattleFeatures 战斗页特征（对应 Lua battle）。
type BattleFeatures struct {
	Feature           vision.Feature
	BackBtn           image.Rectangle
	QuickBattleBtn    vision.FindDef
	QuickDialog       BattleQuickDialog
	SettleBtn         image.Rectangle
	SoulStoneOcr      image.Rectangle
	SoulStones        map[string]map[string]vision.FindDef
	BattleCardFeature vision.FindDef
	PageSwipe         touchSwipe
	LastPageFeature   vision.FindDef
}

// touchSwipe 翻页滑动参数（对应 Lua 翻页滑动表）。
type touchSwipe struct {
	X1, Y1, X2, Y2 int
}

var soulStoneCategories = []string{"史诗", "传奇", "上古", "野兽"}

var battle = func() BattleFeatures {
	var f BattleFeatures
	f.Feature = vision.Feature{Points: "161|84|df958b-101010,730|93|ffffff-101010,821|120|ffffff-101010,1417|78|87433b-101010,1429|799|87433b-101010,102|814|7e763a-101010", Sim: 0.9}
	f.BackBtn = image.Rect(1411, 103, 1422, 114)
	f.QuickBattleBtn = findDef(543, 735, 634, 819, "d3af16-101010",
		"-14|-9|18070a-101010|18|-13|050401-101010|7|27|ffffff-101010|-18|39|463b00-101010|15|-28|493900-101010|-26|-11|ffcf00-101010", 0, 0.9)
	f.QuickDialog = BattleQuickDialog{
		Feature:    vision.Feature{Points: "800|177|d1af19-101010,1111|173|349fdf-101010,523|180|696f9b-101010,876|688|7acd0e-101010,594|768|685408-101010", Sim: 0.9},
		ConfirmBtn: image.Rect(785, 667, 818, 687),
		CancelBtn:  image.Rect(1081, 167, 1096, 183),
		CountOcr:   image.Rect(756, 408, 973, 451),
	}
	f.SettleBtn = image.Rect(745, 821, 858, 876)
	f.SoulStoneOcr = image.Rect(271, 617, 348, 694)
	f.SoulStones = map[string]map[string]vision.FindDef{
		"史诗": {
			"浓缩奶油": soulStone("bba983-101010", "3|-9|9b7370-101010|5|-15|adc1cb-101010|-13|-9|9d59c7-101010|16|12|f9d5a7-101010|-8|11|d789c3-101010|-8|5|dfb187-101010"),
			"牡蛎":   soulStone("cdb9cb-101010", "-8|-8|9ba9d3-101010|11|-12|787fab-101010|-12|-11|855da7-101010|-5|12|eda3b7-101010|17|9|f9dfa7-101010"),
			"雪酪":   soulStone("cdebfb-101010", "3|-14|afc7e7-101010|-13|-14|8564a3-101010|17|0|fbeff3-101010|14|9|fbd7a7-101010|10|-11|cfe3fb-101010|-10|8|d787c3-101010"),
			"辣椒素":  soulStone("f9c36a-101010", "-9|-4|46527c-101010|8|-7|df6a46-101010|12|-6|64565c-101010|8|10|ffe793-101010|-11|10|f77f7a-101010|-15|-3|7751a7-101010"),
			"闪耀之星": soulStone("9dd7f3-101010", "-14|-5|af6bdf-101010|-18|-6|8b58b3-101010|-15|10|c9d5fb-101010|10|7|fdeff3-101010|9|-4|e37faf-101010"),
			"绯红珊瑚": soulStone("e97d8f-101010", "4|-8|7f497e-101010|16|-10|ed78b3-101010|-10|-10|8b5eaf-101010|12|3|e18593-101010|18|8|f9d7a7-101010|-8|10|dd8bc7-101010"),
			"妖精王":  soulStone("f3e5fb-101010", "-10|-13|a9c3e3-101010|-15|-3|7677b3-101010|-16|8|f3e7fb-101010|5|-12|89a7bb-101010|6|5|fbefff-101010"),
			"星辰":   soulStone("997470-101010", "-16|-8|737ddf-101010|7|-6|9587d7-101010|-4|-3|d5e1ef-101010|-11|7|fbf1fb-101010|1|11|d5b1d3-101010|-16|11|d585c7-101010|12|-10|ab6197-101010"),
		},
		"传奇": {
			"雷神武将": soulStone("fddfbb-101010", "3|-9|b1c5af-101010|9|-14|496283-101010|-4|-17|6e9bb3-101010|-12|-12|59a3e3-101010|15|10|ebe3f7-101010|-4|13|9b7fd3-101010"),
			"冰霜女王": soulStone("8799e3-101010", "-6|1|8bafe7-101010|-13|-5|b1c9e7-101010|11|-5|83838f-101010|-14|14|c3cbef-101010|10|15|b9bffb-101010"),
			"海妖精":  soulStone("a1dbeb-101010", "7|0|afddf3-101010|11|-11|d1ddcf-101010|1|-11|4c93d7-101010|-14|-3|719dd3-101010|17|9|b1bdf3-101010|3|-23|7fa9e7-101010"),
			"风箭手":  soulStone("cfe1c7-101010", "-3|-8|7aa160-101010|-16|1|7da164-101010|9|-1|d3d3ab-101010|-16|15|c3c7ef-101010|-5|-17|89aff3-101010"),
		},
		"上古": {
			"莓果": soulStone("c5568f-101010", "12|-9|41897a-101010|-4|-8|4f8f9f-101010|-4|10|ef6ffb-101010|-2|14|9b74d7-101010|16|12|c7a3df-101010"),
		},
		"野兽": {},
	}
	f.BattleCardFeature = findDef(138, 61, 1463, 827, "333333-101010",
		"0|-14|6685bb-101010|-16|0|223858-101010|13|1|4269ab-101010|-1|9|999793-101010|-9|1|999997-101010|7|4|ebebe3-101010", 0, 0.9)
	f.PageSwipe = touchSwipe{X1: 588, Y1: 646, X2: 587, Y2: 150}
	f.LastPageFeature = findDef(599, 549, 1437, 715, "552e2b-101010",
		"767|8|552e2b-101010|774|102|552e2b-101010|412|36|552e2b-101010|220|31|552e2b-101010|707|35|552e2b-101010|71|81|552e2b-101010|188|36|552e2b-101010|21|91|552e2b-101010", 0, 0.9)
	return f
}()

func soulStone(first, offsets string) vision.FindDef {
	return findDef(271, 618, 346, 694, first, offsets, 0, 0.9)
}

// OreVeinCard 矿脉卡找色定义。
type OreVeinCard struct {
	Name string
	Def  vision.FindDef
}

// OreVeinCards 矿脉卡特征（对应 Lua oreVeinCards，键名同配置 miningOreCards）。
var OreVeinCards = map[string]vision.FindDef{
	"flourStone":    oreVeinCard(3, 602, 1587, 707, "dfc3a6-101010", "-10|-11|dfc4a8-101010|-15|-15|502828-101010|13|13|3d1c1b-101010|-17|6|c9ae8f-101010|-22|13|4f2828-101010"),
	"sugarOre":      oreVeinCard(3, 602, 1587, 707, "8d48e7-101010", "0|3|545883-101010|5|-11|aea2fb-101010|-4|-21|fafdfd-101010|5|-23|2085f7-101010|19|-11|46b5fe-101010|-12|2|a4dffd-101010"),
	"butterAmber":   oreVeinCard(15, 542, 1599, 656, "f9fec5-101010", "2|-16|e7c028-101010|-18|2|dc6712-101010|17|2|cc4f0b-101010|-10|14|f1fb92-101010|12|14|5f1310-101010|-10|-11|e6fee8-101010"),
	"amberFossil":   oreVeinCard(3, 602, 1587, 707, "feb96b-101010", "4|-8|a64000-101010|1|10|c49a67-101010|12|-11|76000d-101010|-12|-12|efdea8-101010|17|9|835437-101010|-23|11|5a2614-101010"),
	"purpleFossil":  oreVeinCard(3, 602, 1587, 707, "ea9aff-101010", "4|-8|8000bc-101010|9|-13|a3069e-101010|2|10|6f5ca6-101010|-11|13|bfaee9-101010|16|10|4f3e76-101010|-22|-2|372d53-101010"),
	"emeraldFossil": oreVeinCard(3, 602, 1587, 707, "9ff9c5-101010", "4|-8|1c9145-101010|5|-17|16572f-101010|-18|4|91a98d-101010|0|9|89a188-101010|17|9|536c5b-101010|12|17|07220d-101010"),
}

func oreVeinCard(x1, y1, x2, y2 int, first, offsets string) vision.FindDef {
	return findDef(x1, y1, x2, y2, first, offsets, 0, 0.9)
}

// JellyConfig 配置洋菜冻界面特征。
type JellyConfig struct {
	Feature    vision.Feature
	BackBtn    image.Rectangle
	ChooseBtn  image.Rectangle
	Chooseable vision.Feature
}

// JellyFeatures 解除洋菜冻页特征（对应 Lua 解除洋菜冻_特征）。
type JellyFeatures struct {
	Feature         vision.Feature
	BackBtn         image.Rectangle
	ClaimAllFeature vision.Feature
	ClaimAllBtn     image.Rectangle
	SettleBtn       image.Rectangle
	OcrRegion       image.Rectangle
	Config          JellyConfig
}

var jelly = func() JellyFeatures {
	var f JellyFeatures
	f.Feature = vision.Feature{Points: "246|106|df958b-101010,254|789|87433b-101010,717|114|ffffff-101010,766|145|ffffff-101010,751|148|190c0b-101010,1348|802|622620-101010", Sim: 0.9}
	f.BackBtn = image.Rect(1330, 125, 1338, 141)
	f.ClaimAllFeature = vision.Feature{Points: "1298|759|7acd10-101010", Sim: 0.9}
	f.ClaimAllBtn = image.Rect(1179, 733, 1219, 763)
	f.SettleBtn = image.Rect(699, 758, 907, 806)
	f.OcrRegion = image.Rect(274, 586, 1328, 669)
	f.Config = JellyConfig{
		Feature:    vision.Feature{Points: "712|158|ffffff-101010,885|185|ffffff-101010,717|116|7f7f7e-101010,488|149|df958b-101010,243|107|6f4a44-101010,473|723|87433b-101010,243|745|43211d-101010", Sim: 0.9},
		BackBtn:    image.Rect(1083, 170, 1098, 184),
		ChooseBtn:  image.Rect(973, 700, 995, 719),
		Chooseable: vision.Feature{Points: "1052|731|7acd0e-101010,902|697|93d73e-101010", Sim: 0.9},
	}
	return f
}()

// MineHome 返回矿山首页特征（对应 MineFeatureLib.mineHome()）。
func MineHome() MineHomeFeatures { return mineHome }

// MineVenture 返回勘查域特征（对应 MineFeatureLib.mineVenture()）。
func MineVenture() MineVentureFeatures { return mineVenture }

// Mining 返回开采页特征（对应 MineFeatureLib.mining()）。
func Mining() MiningFeatures { return mining }

// Battle 返回战斗页特征（对应 MineFeatureLib.battle()）。
func Battle() BattleFeatures { return battle }

// Jelly 返回解除洋菜冻页特征（对应 MineFeatureLib.jelly()）。
func Jelly() JellyFeatures { return jelly }

// SoulStoneCategories 灵魂石类别顺序（对应 Lua SOUL_STONE_CATEGORIES）。
func SoulStoneCategories() []string { return soulStoneCategories }
