//go:build !windows

package desktop

import "github.com/DaisyCloverSoftware/workbench/internal/core"

func ensureRunnerChatActivityMonitor(_ *core.Engine) {}

func runnerChatActivitySnapshot() []core.RunnerChatActivityInfo { return nil }
