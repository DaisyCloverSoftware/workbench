package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionWorkSettingsPaintDoesNotSynchronouslyReenterChildren(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate desktop source")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(here), "work_settings_panels_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "procUpdateWindow.Call(hwnd)") {
		t.Fatal("production parent paint synchronously forces child UpdateWindow and can starve the Win32 message pump")
	}
	if !strings.Contains(text, "procInvalidateRect.Call(hwnd, 0, 1)") {
		t.Fatal("production parent paint no longer schedules child repaint through normal message processing")
	}
}
