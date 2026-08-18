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
					activity := filterRunnerChatActivityToInventory(response.Projects, response.ChatActivity)
					runnerChatActivityCache.Store(activity)
					// ChatGPT is the normal entry point. If it has genuinely used a
					// current runner project, register that project in the desktop
					// automatically rather than making the user import the same repo by
					// hand. Activity for review worktrees or vanished runner projects is
					// filtered before it can become user-facing state.
					_, _ = registerActiveChatProjects(eng, response.Projects, activity, time.Now().UTC())
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
