package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProductionWin32LoopPinsItsOSThread(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate desktop source")
	}
	path := filepath.Join(filepath.Dir(here), "production_shell_windows.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	lock := strings.Index(text, "runtime.LockOSThread()")
	create := strings.Index(text, "procCreateWindowExW.Call(")
	pump := strings.Index(text, "procGetMessageW.Call(")
	unlock := strings.Index(text, "defer runtime.UnlockOSThread()")
	if lock < 0 || unlock < 0 {
		t.Fatal("production Win32 window lifetime is not pinned with runtime.LockOSThread/UnlockOSThread")
	}
	if create < 0 || pump < 0 {
		t.Fatal("production Win32 create/message-pump markers are missing")
	}
	if lock > create || lock > pump {
		t.Fatal("OS thread must be locked before creating the HWND and entering its message pump")
	}
}
