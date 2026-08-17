//go:build !windows

package desktop

import "github.com/DaisyCloverSoftware/workbench/internal/core"

func currentRunnerChatBridge() *core.RunnerChatBridgeInfo {
	return nil
}
