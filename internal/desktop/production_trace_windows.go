//go:build windows

package desktop

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var (
	productionTracePath           string
	productionTraceDispatchHWND   atomic.Uint64
	productionTraceDispatchMsg    atomic.Uint32
	productionTraceDispatchWParam atomic.Uint64
	productionTraceDispatchStart  atomic.Int64
	productionTraceParentMsg      atomic.Uint32
	productionTraceParentWParam   atomic.Uint64
	productionTraceParentStart    atomic.Int64
)

// startProductionUITrace is intentionally inert in normal builds. CI can set
// WORKBENCH_UI_TRACE_PATH to obtain a tiny external heartbeat identifying the
// Win32 message currently occupying the UI thread when a watchdog fires.
func startProductionUITrace() {
	productionTracePath = strings.TrimSpace(os.Getenv("WORKBENCH_UI_TRACE_PATH"))
	if productionTracePath == "" {
		return
	}
	go func(path string) {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for now := range ticker.C {
			dispatchStart := productionTraceDispatchStart.Load()
			parentStart := productionTraceParentStart.Load()
			dispatchElapsed := time.Duration(0)
			if dispatchStart > 0 {
				dispatchElapsed = now.Sub(time.Unix(0, dispatchStart))
			}
			parentElapsed := time.Duration(0)
			if parentStart > 0 {
				parentElapsed = now.Sub(time.Unix(0, parentStart))
			}
			content := fmt.Sprintf(
				"timestamp=%s\ndispatch_hwnd=0x%x\ndispatch_message=0x%04x\ndispatch_wparam=0x%x\ndispatch_elapsed=%s\nparent_message=0x%04x\nparent_wparam=0x%x\nparent_elapsed=%s\n",
				now.UTC().Format(time.RFC3339Nano),
				productionTraceDispatchHWND.Load(),
				productionTraceDispatchMsg.Load(),
				productionTraceDispatchWParam.Load(),
				dispatchElapsed,
				productionTraceParentMsg.Load(),
				productionTraceParentWParam.Load(),
				parentElapsed,
			)
			_ = os.WriteFile(path, []byte(content), 0o600)
		}
	}(productionTracePath)
}

func beginProductionDispatchTrace(hwnd uintptr, message uint32, wParam uintptr) func() {
	if productionTracePath == "" {
		return func() {}
	}
	productionTraceDispatchHWND.Store(uint64(hwnd))
	productionTraceDispatchMsg.Store(message)
	productionTraceDispatchWParam.Store(uint64(wParam))
	productionTraceDispatchStart.Store(time.Now().UnixNano())
	return func() {
		productionTraceDispatchStart.Store(0)
		productionTraceDispatchWParam.Store(0)
		productionTraceDispatchMsg.Store(0)
		productionTraceDispatchHWND.Store(0)
	}
}

func beginProductionParentTrace(message uint32, wParam uintptr) func() {
	if productionTracePath == "" {
		return func() {}
	}
	// These are the parent messages capable of doing substantive Workbench work.
	// Colour/draw callbacks are deliberately excluded so nested native-control
	// callbacks do not overwrite the outer operation we are diagnosing.
	switch message {
	case wmCreate, wmSize, wmPaint, wmAppRefresh, wmCommand:
	default:
		return func() {}
	}
	productionTraceParentMsg.Store(message)
	productionTraceParentWParam.Store(uint64(wParam))
	productionTraceParentStart.Store(time.Now().UnixNano())
	return func() {
		productionTraceParentStart.Store(0)
		productionTraceParentWParam.Store(0)
		productionTraceParentMsg.Store(0)
	}
}
