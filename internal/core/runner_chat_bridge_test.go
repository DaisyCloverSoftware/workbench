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
			if tc.name == "private" {
				for _, want := range []string{"safe hands", "direct machine control ready", "explicit owner-authorized OpenClaw handoff available"} {
					if !strings.Contains(got.Status, want) {
						t.Fatalf("private relay status missing %q: %q", want, got.Status)
					}
				}
				if strings.Contains(got.Status, "autonomous handoff ready") {
					t.Fatalf("private relay health must not present autonomous handoff as implicit authority: %q", got.Status)
				}
			}
		})
	}
}
