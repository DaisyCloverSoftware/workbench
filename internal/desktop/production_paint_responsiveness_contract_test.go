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
		"rdwUpdateNow   = 0x0100",
		"rdwAllChildren = 0x0080",
		"rdwInvalidate|rdwErase|rdwAllChildren",
	} {
		if strings.Contains(visualText, forbidden) {
			t.Fatalf("production redraw still invalidates the full child tree or forces a synchronous repaint: %q", forbidden)
		}
	}
	if !strings.Contains(visualText, "procRedrawWindow.Call(hwnd, 0, 0, rdwInvalidate|rdwErase)") {
		t.Fatal("production redraw no longer queues the parent surface through the normal message loop")
	}
	for _, chromeID := range []string{"idNavDashboard", "idNavWork", "idNavSettings", "idTopNewTask", "idTopNeedsYou", "idTopReview"} {
		if !strings.Contains(visualText, chromeID) {
			t.Fatalf("production redraw no longer refreshes owner-drawn chrome %s", chromeID)
		}
	}
}
