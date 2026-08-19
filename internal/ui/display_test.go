package ui

import "testing"

func TestPanelProjectsDisplayProfile(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{
		Settings: Default(),
		Display:  DisplayProfile{Width: 1280, Height: 720, RequiredWidth: 1600, RequiredHeight: 900},
	}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()
	frame, ok := panel.readFrame()
	if !ok || frame.Display.Width != 1280 || frame.Display.Height != 720 {
		t.Fatalf("display not projected: %+v", frame.Display)
	}
	if err := panel.PublishDisplay(DefaultDisplayProfile()); err != nil {
		t.Fatal(err)
	}
	frame, ok = panel.readFrame()
	if !ok || frame.Display.Width != 1600 {
		t.Fatalf("publish display lost: %+v", frame.Display)
	}
}

func TestWriteFrameDoesNotClobberDisplayProfile(t *testing.T) {
	panel := NewPanel()
	if err := panel.Open(Snapshot{Settings: Default(), Display: DefaultDisplayProfile()}, func(Command) {}); err != nil {
		t.Fatal(err)
	}
	defer panel.Close()
	frame, _ := panel.readFrame()
	_ = panel.PublishDisplay(DisplayProfile{Width: 1, Height: 1, RequiredWidth: 1600, RequiredHeight: 900})
	panel.writeFrame(frame)
	next, _ := panel.readFrame()
	if next.Display.Width != 1 {
		t.Fatalf("writeFrame overwrote display: %+v", next.Display)
	}
}
