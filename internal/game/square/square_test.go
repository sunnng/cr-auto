package square

import (
	"image"
	"path/filepath"
	"sync"
	"testing"
	"time"

	libcolor "app/internal/lib/color"
	"app/internal/lib/ocr"
	"app/internal/lib/store"
	"app/internal/lib/touch"
)

type touchRecorder struct {
	mu     sync.Mutex
	points []image.Point
}

func (t *touchRecorder) tap(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.points = append(t.points, image.Point{X: x, Y: y})
}

func (t *touchRecorder) taps() []image.Point {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]image.Point(nil), t.points...)
}

type fakeOcr struct {
	byRect map[image.Rectangle]string
}

// Scan 返回按 returnType 适配的假识别结果：num/text 返回存值本身，json 返回词条包。
func (f *fakeOcr) Scan(rect image.Rectangle, mode int, returnType string) (string, error) {
	if raw, ok := f.byRect[rect]; ok {
		if returnType == "json" {
			return `[{"words":"` + raw + `","location":[[0,0],[10,0],[10,10],[0,10]]}]`, nil
		}
		return raw, nil
	}
	return "", nil
}

func (f *fakeOcr) FindTapPoint(text string, rect image.Rectangle) (int, int, bool) {
	return 0, 0, false
}

func setupTest(t *testing.T, hits libcolor.Screen, eng *fakeOcr) *touchRecorder {
	t.Helper()
	rec := &touchRecorder{}
	if hits == nil {
		hits = libcolor.NewScriptedScreen()
	}
	libcolor.SetScreen(hits)
	libcolor.SetSleep(func(ms int) {})
	touch.SetPerform(touch.Perform{
		Tap:    rec.tap,
		Random: func(min, max int) int { return 0 },
		Sleep:  func(ms int) {},
	})
	store.SetDefault(store.New(filepath.Join(t.TempDir(), "store.json")))
	if eng == nil {
		eng = &fakeOcr{byRect: map[image.Rectangle]string{}}
	}
	ocr.SetEngine(eng)
	t.Cleanup(func() {
		libcolor.SetScreen(nil)
		libcolor.SetSleep(nil)
		touch.SetPerform(touch.Perform{})
		store.SetDefault(nil)
		ocr.SetEngine(nil)
	})
	return rec
}

// fixedNow 可推进的假时钟。
type fixedNow struct {
	t time.Time
}

func (f *fixedNow) advance(d time.Duration) {
	f.t = f.t.Add(d)
}

// ============ 会话测试 ============

func TestSessionDoneTodayLifecycle(t *testing.T) {
	setupTest(t, nil, nil)
	if IsDoneToday() {
		t.Fatal("fresh store must not be done today")
	}
	MarkDoneToday()
	if !IsDoneToday() {
		t.Fatal("markDoneToday must set today's date")
	}
	ClearAll()
	if IsDoneToday() {
		t.Fatal("clearAll must reset done date")
	}
}

func TestSessionStayAccumulation(t *testing.T) {
	setupTest(t, nil, nil)
	clock := &fixedNow{t: time.Now()}
	SetNow(func() time.Time { return clock.t })
	defer SetNow(time.Now)

	Ensure()
	if StayElapsed() != 0 {
		t.Fatalf("fresh session elapsed=%d want 0", StayElapsed())
	}
	StartStay()
	clock.advance(30 * time.Second)
	elapsed := StayElapsed()
	if elapsed != 30 {
		t.Fatalf("startStay must accumulate: elapsed=%d want 30", elapsed)
	}
	PauseStay()
	frozen := StayElapsed()
	if frozen != 30 {
		t.Fatalf("pauseStay must settle accumulated seconds: got %d want 30", frozen)
	}
	// 暂停后不再累积。
	clock.advance(10 * time.Second)
	if StayElapsed() != frozen {
		t.Fatalf("pauseStay must freeze elapsed: %d != %d", StayElapsed(), frozen)
	}
	// reset 清零并重新计时。
	ResetStayTimer()
	if rem := StayRemaining(60); rem != 60 {
		t.Fatalf("stayRemaining after reset=%d want 60", rem)
	}
	clock.advance(5 * time.Second)
	if rem := StayRemaining(60); rem != 55 {
		t.Fatalf("stayRemaining after 5s=%d want 55", rem)
	}
}

func TestSessionCheckedToday(t *testing.T) {
	setupTest(t, nil, nil)
	if HasCheckedToday() {
		t.Fatal("fresh session must not be checked today")
	}
	MarkCheckedToday()
	if !HasCheckedToday() {
		t.Fatal("markCheckedToday must set checkedDate")
	}
}

// ============ 页面测试 ============

func TestSquarePageDetection(t *testing.T) {
	features := FeatureLib()
	setupTest(t, libcolor.HitFeatures(features.Home.Feature), nil)
	if !IsCurrent() {
		t.Fatal("home feature must be detected")
	}
	setupTest(t, libcolor.HitFeatures(features.DialogLeave.Feature), nil)
	if !IsLeaveDialog() {
		t.Fatal("dialog feature must be detected")
	}
}

func TestReadRewardSum(t *testing.T) {
	features := FeatureLib()
	eng := &fakeOcr{byRect: map[image.Rectangle]string{
		features.DialogLeave.RewardNowOcr:   "120",
		features.DialogLeave.RewardTotalOcr: "80",
	}}
	setupTest(t, libcolor.HitFeatures(features.DialogLeave.Feature), eng)
	pending, total, sum, ok := ReadRewardSum()
	if !ok || pending != 120 || total != 80 || sum != 200 {
		t.Fatalf("reward sum=(%d,%d,%d) ok=%v want (120,80,200)", pending, total, sum, ok)
	}
}

func TestTextIndicatesMaxed(t *testing.T) {
	cases := map[string]bool{
		"已到达最大等级":   true,
		"今日奖励已领取完毕": true,
		"奖励未领取":     false,
		"":          false,
	}
	for in, want := range cases {
		if got := textIndicatesMaxed(in); got != want {
			t.Fatalf("textIndicatesMaxed(%q)=%v want %v", in, got, want)
		}
	}
}
