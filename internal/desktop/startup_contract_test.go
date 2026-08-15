package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionEntryUsesFailClosedOwnershipAwareStartup(t *testing.T) {
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
	if !strings.Contains(text, "workbenchSingleInstanceHandle == 0") {
		t.Fatal("production entry does not fail closed when named-mutex ownership is unavailable")
	}
	if !strings.Contains(text, "desktop.RunOwned(appVersion, true)") {
		t.Fatal("production entry is not bound to confirmed ownership-aware desktop startup")
	}
	if strings.Contains(text, "desktop.Run(appVersion)") {
		t.Fatal("production entry still calls the legacy unconditional recovery startup")
	}
}
