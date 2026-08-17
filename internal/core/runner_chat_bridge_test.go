package core

import (
	"strings"
	"testing"
)

func TestClassifyRunnerChatBridgeRequiresVerifiedPrivateModeForReady(t *testing.T) {
	cases := []struct {
		name      string
		active    bool
		execStart string
		ready     bool
		transport string
	}{
		{name: "inactive", active: false, ready: false, transport: "git-relay"},
		{name: "private", active: true, execStart: "/bin/workbench-relay --public-transport=false", ready: true, transport: "private-git-relay"},
		{name: "public", active: true, execStart: "/bin/workbench-relay --public-transport=true", ready: false, transport: "public-git-relay"},
		{name: "unknown", active: true, execStart: "/bin/workbench-relay", ready: false, transport: "git-relay"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyRunnerChatBridge(tc.active, tc.execStart)
			if got == nil {
				t.Fatal("bridge classification returned nil")
			}
			if got.Ready != tc.ready || got.Transport != tc.transport {
				t.Fatalf("classification=%+v want ready=%v transport=%q", got, tc.ready, tc.transport)
			}
			if got.Status == "" {
				t.Fatal("bridge classification must provide a categorical status")
			}
			if tc.name == "private" && (!strings.Contains(got.Status, "safe hands") || !strings.Contains(got.Status, "autonomous handoff")) {
				t.Fatalf("private relay status must describe both Chat-first paths: %q", got.Status)
			}
		})
	}
}
