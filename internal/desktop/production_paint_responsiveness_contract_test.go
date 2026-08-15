package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionPaintDoesNotCreateSynchronousChildRepaintStorms(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate desktop source")
	}
	dir := filepath.Dir(here)
	panels, err := os.ReadFile(filepath.Join(dir, "work_settings_panels_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	panelText := string(panels)
	for _, forbidden := range []string{"procUpdateWindow.Call(hwnd)", "procInvalidateRect.Call(hwnd, 0, 1)"} {
		if strings.Contains(panelText, forbidden) {
			t.Fatalf("production parent paint still schedules child repaint work: %q", forbidden)
		}
	}

	visual, err := os.ReadFile(filepath.Join(dir, "visual_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	visualText := string(visual)
	for _, forbidden := range []string{
		"rdwInvalidate|rdwErase|rdwAllChildren|rdwUpdateNow",
		"rdwInvalidate | rdwErase | rdwAllChildren | rdwUpdateNow",
		"rdwUpdateNow   = 0x0100",
	} {
		if strings.Contains(visualText, forbidden) {
			t.Fatalf("production redraw path still forces synchronous UpdateNow: %q", forbidden)
		}
	}
	if !strings.Contains(visualText, "rdwInvalidate|rdwErase|rdwAllChildren") {
		t.Fatal("production redraw no longer queues parent and child invalidation through the normal message loop")
	}
}
