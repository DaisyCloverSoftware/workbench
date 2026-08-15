package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionEntryUsesOwnershipAwareDesktopStartup(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	path := filepath.Join(filepath.Dir(here), "..", "..", "cmd", "workbench", "main_windows.go")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "workbenchSingleInstanceHandle != 0") {
		t.Fatal("production entry does not pass the named-mutex ownership proof")
	}
	if !strings.Contains(text, "desktop.RunOwned(appVersion, processOwnershipConfirmed)") {
		t.Fatal("production entry is not bound to ownership-aware desktop startup")
	}
	if strings.Contains(text, "desktop.Run(appVersion)") {
		t.Fatal("production entry still calls the legacy unconditional recovery startup")
	}
}
