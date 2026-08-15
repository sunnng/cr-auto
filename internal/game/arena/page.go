// Package arena 对应 Lua 工程的 game/常规_王国竞技场/：竞技场特征库、页面、会话、路由与任务。
package arena

import (
	"image"
	"strconv"
	"strings"
	"time"

	"app/internal/config"
	"app/internal/lib/color"
	"app/internal/lib/logger"
	"app/internal/lib/ocr"
	"app/internal/lib/touch"
	"app/internal/lib/utils"
	"app/internal/vision"
)

const pageTag = "[王国竞技场.页面]"

var settlementResults = map[string]bool{
	"胜利": true,
	"失败": true,
	"平局": true,
}

// hasLeaveBtn 结算页离开按钮是否可见（对应 Lua hasLeaveBtn）。
func hasLeaveBtn() bool { return color.Match(arenaFeatures.Settlement.LeaveFeature) }

// ParseBattleResult 结算页结果 OCR 解析（对应 Lua parseBattleResult）。
// 命中胜利/失败/平局即返回结果，否则 ok=false。
func ParseBattleResult(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	text := utils.KeepHanAlphaNum(raw)
	if settlementResults[text] {
		return text, true
	}
	if strings.Contains(text, "胜利") {
		return "胜利", true
	}
	if strings.Contains(text, "失败") {
		return "失败", true
	}
	if strings.Contains(text, "平局") {
		return "平局", true
	}
	return "", false
}

func readBattleResult() (string, string) {
	raw := ocr.RecognizeText(arenaFeatures.Settlement.ResultOcr)
	result, ok := ParseBattleResult(raw)
	if !ok {
		return "", raw
	}
	return result, raw
}

func isSettlement() bool {
	result, _ := readBattleResult()
	return result != ""
}

// waitBattleResult 轮询等待结算页结果（对应 Lua waitBattleResult，默认 120s）。
func waitBattleResult(maxWait, interval int) string {
	if maxWait <= 0 {
		maxWait = 120000
	}
	if interval <= 0 {
		interval = 1000
	}
	deadline := time.Now().UnixMilli() + int64(maxWait)
	for time.Now().UnixMilli() < deadline {
		result, raw := readBattleResult()
		if result != "" {
			logger.Debug(pageTag, " 命中 结算页 OCR=[%s]", raw)
			return result
		}
		color.Sleep(interval, interval)
	}
	logger.Warn(pageTag, " 等待 结算页 超时 %dms", maxWait)
	return ""
}

func waitFeature(feature vision.Feature, maxWait, interval int, label string) bool {
	if maxWait <= 0 {
		maxWait = 30000
	}
	if interval <= 0 {
		interval = 500
	}
	ok, _ := color.Wait(feature, maxWait, interval)
	if ok {
		logger.Debug(pageTag, " 命中 %s", label)
	} else {
		logger.Warn(pageTag, " 等待 %s 超时 %dms", label, maxWait)
	}
	return ok
}

// tapToLobby 连续点击离开按钮直到大厅（对应 Lua leaveSettlement/tapToLobby 防御性编程）。
func tapToLobby() bool {
	if IsLobby() {
		return true
	}
	if !isSettlement() && !hasLeaveBtn() {
		return false
	}
	opts := color.TapOpts{TimeoutMs: 60000, IntervalMs: 500, TapDelayMs: 500, SleepMs: 1200}
	if ok, _ := color.TapUntilMatch(arenaFeatures.Settlement.LeaveBtn, arenaFeatures.Lobby.Feature, opts); !ok {
		return false
	}
	// 防止先到达大厅，再弹出升段页面：再次执行。
	if IsLobby() {
		return true
	}
	ok, _ := color.TapUntilMatch(arenaFeatures.Settlement.LeaveBtn, arenaFeatures.Lobby.Feature, opts)
	return ok
}

// IsLobby 是否在竞技场大厅（对应 ArenaPage.isLobby）。
func IsLobby() bool { return color.Match(arenaFeatures.Lobby.Feature) }

// WaitLobby 等待竞技场大厅出现（对应 ArenaPage.waitLobby，默认 30s）。
func WaitLobby(timeoutMs int) bool {
	return waitFeature(arenaFeatures.Lobby.Feature, timeoutMs, 500, "竞技场大厅")
}

// TapClose 关闭竞技场大厅（对应 ArenaPage.tapClose，默认 1200ms）。
func TapClose(delayMs int) {
	touch.TapArea(arenaFeatures.Lobby.CloseBtn, defaultDelay(delayMs, 1200))
}

// ReadMedalAndTicket 读取勋章与门票（对应 ArenaPage.readMedalAndTicket）。
// 勋章与门票独立解析：medalOk/ticketOk 分别表示各自是否识别成功。
func ReadMedalAndTicket() (medal, ticket int, medalOk, ticketOk bool) {
	words := ocr.RecognizeWords(arenaFeatures.Lobby.MedalTicketOcr)
	if len(words) < 2 {
		return 0, 0, false, false
	}
	medal, medalOk = utils.ParseNumber(words[0])
	if st, sOk := utils.ParseStamina(words[1]); sOk {
		ticket, ticketOk = st.Current, true
	} else if t, tOk := utils.ParseNumber(words[1]); tOk {
		ticket, ticketOk = t, true
	}
	return medal, ticket, medalOk, ticketOk
}

// ReadTrophyCount 读取奖杯数（对应 ArenaPage.readTrophyCount）。
func ReadTrophyCount() (int, bool) {
	raw := ocr.RecognizeText(arenaFeatures.Lobby.TrophyOcr)
	if raw == "" {
		return 0, false
	}
	if st, ok := utils.ParseStamina(raw); ok {
		return st.Current, true
	}
	if n, ok := utils.ParseNumber(raw); ok {
		return n, true
	}
	return 0, false
}

// OpponentInfo 对手信息（对应 Lua readOpponentInfo 返回表）。
type OpponentInfo struct {
	Site         image.Point
	IsBattled    bool
	BattleResult string
	Trophies     int
}

// ReadOpponentInfo 读取指定位置的对手信息（对应 ArenaPage.readOpponentInfo）。
// location 从 1 开始（对应 Lua 数组下标）。
func ReadOpponentInfo(location int) OpponentInfo {
	var info OpponentInfo
	points := color.FindAll(arenaFeatures.Opponent.FindDef)
	if location < 1 || location > len(points) {
		return info
	}
	target := points[location-1]
	info.Site = target

	base := arenaFeatures.Opponent.BaseSite
	tr := arenaFeatures.Opponent.TrophyRect
	trophyRect := utils.GenerateNewPos(target.X, target.Y, base.X, base.Y, tr.Min.X, tr.Min.Y, tr.Max.X, tr.Max.Y)
	if trophies := ocr.RecognizeNumber(trophyRect); trophies != "" {
		if n, ok := utils.ParseNumber(trophies); ok {
			info.Trophies = n
		}
	}

	rx := target.X + arenaFeatures.Opponent.ResultOffset.X
	ry := target.Y + arenaFeatures.Opponent.ResultOffset.Y
	colors := arenaFeatures.Opponent.ResultColors
	switch {
	case color.MatchRGB(rx, ry, colors.Win, 0.95):
		info.IsBattled, info.BattleResult = true, "胜利"
	case color.MatchRGB(rx, ry, colors.Draw, 0.95):
		info.IsBattled, info.BattleResult = true, "平局"
	case color.MatchRGB(rx, ry, colors.Lose, 0.95):
		info.IsBattled, info.BattleResult = true, "失败"
	}
	return info
}

func isClose(a, b, threshold int) bool {
	if threshold == 0 {
		return false
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < threshold
}

// FindFirstValidOpponent 扫描当前页所有对手，返回第一个符合要求的（对应 ArenaPage.findFirstValidOpponent）。
func FindFirstValidOpponent(cfg config.ArenaConfig, myTrophy int) (OpponentInfo, bool) {
	points := color.FindAll(arenaFeatures.Opponent.FindDef)
	if len(points) == 0 {
		return OpponentInfo{}, false
	}
	trophyDiff := cfg.TrophyDiff
	for loc := 1; loc <= len(points); loc++ {
		info := ReadOpponentInfo(loc)
		if info.Site.X == 0 && info.Site.Y == 0 {
			continue
		}
		if info.IsBattled {
			logger.Info(pageTag, " 位%d 已战斗(%s) 跳过", loc, info.BattleResult)
		} else if myTrophy > info.Trophies && !isClose(myTrophy, info.Trophies, trophyDiff) {
			logger.Info(pageTag, " 位%d 奖杯过滤 我方=%d 对手=%d", loc, myTrophy, info.Trophies)
		} else {
			logger.Info(pageTag, " 位%d 可开战  奖杯=%d", loc, info.Trophies)
			return info, true
		}
	}
	return OpponentInfo{}, false
}

// SwipePageLeft 左滑翻页（对应 ArenaPage.swipePageLeft）。
func SwipePageLeft() {
	s := arenaFeatures.Pagination.SwipeLeft
	touch.SwipeX(s.X1, s.X2, s.Y1, touch.SwipeOpts{MoveMs: 500, HoldMs: 200})
	color.Sleep(1000, 500)
}

// RunBattle 开战并等待结算（对应 ArenaPage.runBattle）。
// 返回 "胜利"/"失败"/"平局"；失败返回 ""。
func RunBattle() string {
	logger.Info(pageTag, " 等待队伍选择页")
	if !waitFeature(arenaFeatures.TeamSelect.Feature, 30000, 500, "队伍选择") {
		return ""
	}

	logger.Info(pageTag, " 点击开始战斗")
	start := arenaFeatures.TeamSelect.StartBattle
	touch.TapR(start.X, start.Y, 1000)

	if color.Match(arenaFeatures.Dialog.DeployMore.Feature) {
		confirm := arenaFeatures.Dialog.DeployMore.Confirm
		touch.TapR(confirm.X, confirm.Y, 1000)
	}
	if color.Match(arenaFeatures.Dialog.MissingTopping.Feature) {
		confirm := arenaFeatures.Dialog.MissingTopping.Confirm
		touch.TapR(confirm.X, confirm.Y, 0)
	}

	logger.Info(pageTag, " 等待队伍选择页消失")
	if !color.WaitGone(arenaFeatures.TeamSelect.Feature, 30000, 500) {
		logger.Warn(pageTag, " 队伍选择页未消失，可能未进入战斗")
		return ""
	}

	logger.Info(pageTag, " 已进入战斗，等待战斗结果")
	result := waitBattleResult(120000, 1000)
	if result == "" {
		logger.Warn(pageTag, " 未等到结算页")
		return ""
	}

	logger.Info(pageTag, " 战斗结果: %s", result)
	color.Sleep(1500, 500)

	if !tapToLobby() && !IsLobby() {
		logger.Warn(pageTag, " 离开结算失败")
		return ""
	}
	return result
}

// IsFreeRefresh 是否可免费刷新（对应 ArenaPage.isFreeRefresh）。
func IsFreeRefresh() bool {
	text := ocr.RecognizeText(arenaFeatures.Lobby.FreeRefreshOcr)
	return text == "免费刷新"
}

// TapFreeRefresh 点击免费刷新（对应 ArenaPage.tapFreeRefresh）。
func TapFreeRefresh() {
	tap := arenaFeatures.Lobby.FreeRefreshTap
	touch.TapR(tap.X, tap.Y, 1000)
}

// ReadRefreshCountdown 解析免费刷新倒计时（对应 ArenaPage.readRefreshCountdown）。
func ReadRefreshCountdown() (int, bool) {
	raw := ocr.RecognizeText(arenaFeatures.Lobby.RefreshOcr)
	if raw == "" {
		return 0, false
	}
	text := utils.KeepHanAlphaNum(raw)
	logger.Debug(pageTag, " 刷新倒计时原始OCR=[%s] 过滤后=[%s]", raw, text)

	var numbers []int
	for _, part := range strings.FieldsFunc(text, func(r rune) bool {
		return r < '0' || r > '9'
	}) {
		if n, err := strconv.Atoi(part); err == nil {
			numbers = append(numbers, n)
		}
	}

	hasMin := strings.Contains(text, "分")
	hasSec := strings.Contains(text, "秒")

	if len(numbers) == 0 {
		logger.Warn(pageTag, " 未能解析倒计时: %s", text)
		return 0, false
	}

	// 同时出现「分」和「秒」：取前两个数字分别作为分、秒。
	if hasMin && hasSec {
		if len(numbers) >= 2 {
			return numbers[0]*60 + numbers[1], true
		}
		logger.Warn(pageTag, " 有分秒关键字但只有一个数字: %s", text)
		return numbers[0] * 60, true
	}
	// 只有「秒」。
	if hasSec {
		return numbers[0], true
	}
	// 只有「分」。
	if hasMin {
		logger.Warn(pageTag, " 只解析到分钟，秒数缺失: %s", text)
		return numbers[0] * 60, true
	}

	// 兜底：冒号格式 mm:ss（作用于过滤后文本；与 Lua 一致，冒号已被剔除时不会命中）。
	if idx := strings.Index(text, ":"); idx > 0 && strings.Contains(text[idx+1:], ":") {
		parts := strings.Split(text[idx-1:], ":")
		mm, err1 := strconv.Atoi(parts[0])
		ss, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			return mm*60 + ss, true
		}
	}

	logger.Warn(pageTag, " 未能解析倒计时: %s", text)
	return 0, false
}

// BuyTicket 打开买票弹窗并确认（对应 ArenaPage.buyTicket）。
func BuyTicket() {
	logger.Info(pageTag, " 打开买票弹窗")
	btn := arenaFeatures.Lobby.BuyTicketBtn
	touch.TapR(btn.X, btn.Y, 1500)

	logger.Debug(pageTag, " 拖动买票滑条")
	s := arenaFeatures.Lobby.BuyTicketSlider
	touch.SwipeEx(touch.SwipeOpts{
		X1: s.Min.X, Y1: s.Min.Y, X2: s.Max.X, Y2: s.Max.Y,
		MoveMs: 1000, HoldMs: 200, DownMs: 50,
	})
	color.Sleep(1000, 500)

	confirm := arenaFeatures.Lobby.BuyTicketConfirm
	touch.TapR(confirm.X, confirm.Y, 0)
	logger.Info(pageTag, " 买票确认已点击")
}

func defaultDelay(delayMs, fallback int) int {
	if delayMs <= 0 {
		return fallback
	}
	return delayMs
}
