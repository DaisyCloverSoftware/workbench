//go:build windows

package desktop

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

var runnerChatActivityCache atomic.Value
var runnerChatActivityMonitorStarted atomic.Bool

func ensureRunnerChatActivityMonitor(eng *core.Engine) {
	if eng == nil || !runnerChatActivityMonitorStarted.CompareAndSwap(false, true) {
		return
	}
	go func() {
		for {
			host := strings.TrimSpace(eng.State().Preferences.OpenClawSSHHost)
			if host == "" {
				runnerChatActivityCache.Store([]core.RunnerChatActivityInfo{})
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
				response, err := core.RunRunnerToolSSH(ctx, host, core.RunnerToolRequest{Action: "chat_activity", Limit: 100})
				cancel()
				if err == nil {
					// Keep the complete privacy-safe relay inventory for the operations
					// board. Server and Windows controls intentionally do not carry a
					// project ref, so filtering the cache to project inventory made real
					// work disappear from the dashboard.
					runnerChatActivityCache.Store(append([]core.RunnerChatActivityInfo(nil), response.ChatActivity...))
					// Project auto-registration remains stricter: only activity tied to
					// a currently advertised runner project may create desktop project
					// entries.
					projectActivity := filterRunnerChatActivityToInventory(response.Projects, response.ChatActivity)
					_, _ = registerActiveChatProjects(eng, response.Projects, projectActivity, time.Now().UTC())
					pruneUnavailableRunnerProjects(eng, response.Projects)
				}
			}
			if runningShell != nil && runningShell.hwnd != 0 {
				procPostMessageW.Call(runningShell.hwnd, wmAppRefresh, 0, 0)
			}
			time.Sleep(15 * time.Second)
		}
	}()
}

func runnerChatActivitySnapshot() []core.RunnerChatActivityInfo {
	value := runnerChatActivityCache.Load()
	if value == nil {
		return nil
	}
	items, ok := value.([]core.RunnerChatActivityInfo)
	if !ok {
		return nil
	}
	return append([]core.RunnerChatActivityInfo(nil), items...)
}
