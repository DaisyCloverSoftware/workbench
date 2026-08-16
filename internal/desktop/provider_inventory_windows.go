//go:build windows

package desktop

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const runnerProviderCacheTTL = 30 * time.Second

var runnerProviderGeneration atomic.Uint64
var loadedSettingsRunnerProviderGeneration uint64

var runnerProviderCache = struct {
	sync.RWMutex
	host      string
	providers []core.RunnerProviderInfo
	loading   bool
	failed    bool
	updatedAt time.Time
}{}

type runnerProviderCacheView struct {
	Providers []core.RunnerProviderInfo
	Loading   bool
	Failed    bool
}

func resetRunnerProviderInventory() {
	runnerProviderCache.Lock()
	runnerProviderCache.host = ""
	runnerProviderCache.providers = nil
	runnerProviderCache.loading = false
	runnerProviderCache.failed = false
	runnerProviderCache.updatedAt = time.Time{}
	runnerProviderCache.Unlock()
	runnerProviderGeneration.Add(1)
}

func runnerProviderInventory(host string) runnerProviderCacheView {
	host = strings.TrimSpace(host)
	runnerProviderCache.RLock()
	defer runnerProviderCache.RUnlock()
	if host == "" || runnerProviderCache.host != host {
		return runnerProviderCacheView{}
	}
	return runnerProviderCacheView{
		Providers: append([]core.RunnerProviderInfo(nil), runnerProviderCache.providers...),
		Loading:   runnerProviderCache.loading,
		Failed:    runnerProviderCache.failed,
	}
}

func (s *Shell) ensureRunnerProviderInventory(force bool) {
	host := strings.TrimSpace(s.eng.State().Preferences.OpenClawSSHHost)
	if host == "" || s.hwnd == 0 {
		return
	}
	now := time.Now()
	runnerProviderCache.Lock()
	if runnerProviderCache.host != host {
		runnerProviderCache.host = host
		runnerProviderCache.providers = nil
		runnerProviderCache.loading = false
		runnerProviderCache.failed = false
		runnerProviderCache.updatedAt = time.Time{}
	}
	if runnerProviderCache.loading {
		runnerProviderCache.Unlock()
		return
	}
	if !force && !runnerProviderCache.updatedAt.IsZero() && now.Sub(runnerProviderCache.updatedAt) < runnerProviderCacheTTL {
		runnerProviderCache.Unlock()
		return
	}
	runnerProviderCache.loading = true
	runnerProviderCache.failed = false
	runnerProviderCache.Unlock()

	go func(requestHost string, allowVerifiedUpgrade bool) {
		fetch := func() (core.RunnerToolResponse, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			return core.RunRunnerToolSSH(ctx, requestHost, core.RunnerToolRequest{Action: "list_providers"})
		}
		response, err := fetch()

		// Rescan diagnoses unattended SSH but never opens a surprise interactive
		// console. The operator explicitly selects the Runner SSH connection row
		// and presses Connect; after completing that console flow, Rescan proves
		// that the same background BatchMode transport now succeeds.
		if err != nil && allowVerifiedUpgrade && runnerInventoryProtocolTooOld(err) {
			// Rescan may use the already-existing verified cluster updater to bring
			// an older runner to the current stable protocol, then retry. Merely
			// opening Settings never mutates the runner.
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 15*time.Minute)
			_, updateErr := core.UpdateWorkbenchRunnerSSH(updateCtx, requestHost)
			updateCancel()
			if updateErr == nil {
				response, err = fetch()
			}
		}

		runnerProviderCache.Lock()
		if runnerProviderCache.host != requestHost {
			runnerProviderCache.Unlock()
			return
		}
		runnerProviderCache.loading = false
		runnerProviderCache.updatedAt = time.Now()
		if err != nil {
			// Preserve only a privacy-safe categorical transport reason. This
			// deliberately does not retain stderr, account details, command output,
			// key paths or other authentication material.
			runnerProviderCache.providers = []core.RunnerProviderInfo{runnerConnectionProviderInfo(err)}
			runnerProviderCache.failed = true
		} else {
			runnerProviderCache.providers = append([]core.RunnerProviderInfo(nil), response.Providers...)
			runnerProviderCache.failed = false
		}
		runnerProviderCache.Unlock()
		runnerProviderGeneration.Add(1)
		if s.hwnd != 0 {
			procPostMessageW.Call(s.hwnd, wmAppRefresh, 0, 0)
		}
	}(host, force)
}

func runnerConnectionProviderInfo(err error) core.RunnerProviderInfo {
	status := "SSH connection failed"
	switch {
	case errors.Is(err, core.ErrRunnerSSHAuthentication):
		status = "unattended SSH authentication required · select and Connect, then Rescan"
	case errors.Is(err, core.ErrRunnerSSHClientUnavailable):
		status = "Windows OpenSSH client unavailable"
	case errors.Is(err, core.ErrRunnerSSHConnectionTimeout):
		status = "SSH connection timed out"
	case errors.Is(err, core.ErrRunnerExecutableUnavailable):
		status = "Workbench Runner executable unavailable on host"
	case errors.Is(err, core.ErrRunnerSSHTransportUnavailable):
		status = "SSH transport unavailable"
	}
	return core.RunnerProviderInfo{
		ID:            core.RunnerConnectionProviderID,
		Name:          "Runner SSH connection",
		Capability:    "unattended Workbench Runner SSH control transport",
		Status:        status,
		Cost:          core.CostIncluded,
		Installed:     true,
		Authenticated: false,
		Ready:         false,
	}
}

func runnerInventoryProtocolTooOld(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(text, "unsupported runner tool action") || strings.Contains(text, "runner tool transport returned invalid json")
}

func runnerProviderByID(host, id string) (core.RunnerProviderInfo, bool) {
	for _, provider := range runnerProviderInventory(host).Providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return core.RunnerProviderInfo{}, false
}

func providerListTarget(raw string) (scope, id string) {
	raw = strings.TrimSpace(raw)
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return "", raw
	}
	return parts[0], parts[1]
}
