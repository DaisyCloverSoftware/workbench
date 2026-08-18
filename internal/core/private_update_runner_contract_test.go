package core

import (
	"os"
	"strings"
	"testing"
)

func TestPrivateLoopUpdateRefreshesSSHRunnerBeforeMCPAndRelay(t *testing.T) {
	b, err := os.ReadFile("../../scripts/bootstrap-private-relay.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	runner := strings.Index(text, `bash "$update_source_dir/scripts/install-runner.sh"`)
	mcp := strings.Index(text, `bash "$update_source_dir/scripts/install-cluster-mcp.sh"`)
	relay := strings.Index(text, `bash "$update_source_dir/scripts/install-github-relay.sh"`)
	if runner < 0 || mcp < 0 || relay < 0 {
		t.Fatalf("private update is missing a required installer: runner=%d mcp=%d relay=%d", runner, mcp, relay)
	}
	if !(runner < mcp && mcp < relay) {
		t.Fatalf("private update must refresh runner -> MCP -> relay from one source: runner=%d mcp=%d relay=%d", runner, mcp, relay)
	}
}
