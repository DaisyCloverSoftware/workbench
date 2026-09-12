package core

import (
	"os"
	"strings"
	"testing"
)

func TestHostBridgeTargetSourceWindowsWiring(t *testing.T) {
	raw, err := os.ReadFile("host_bridge_agent_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	for _, text := range []string{
		"RunWindowsHostBridgeAgentWithTargetSource",
		"validateSSHHostTarget(hostSource())",
		"runHostBridgeTargetLoop(ctx,",
		"RunHostBridgeRPCSSH(pollCtx, sshHost,",
		"RunHostBridgeRPCSSH(completeCtx, sshHost,",
		"windowsHostBridgePollInterval)",
	} {
		if !strings.Contains(source, text) {
			t.Fatalf("missing target-source wiring: %s", text)
		}
	}
	if strings.Count(source, "hostSource()") != 1 {
		t.Fatal("saved target must be sampled only between cycles, never during job completion")
	}
}
