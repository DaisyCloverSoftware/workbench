package desktop

import "testing"

func TestCanRecoverInterruptedTasksRequiresOwnershipProof(t *testing.T) {
	cases := []struct {
		name    string
		process bool
		mcp     bool
		want    bool
	}{
		{name: "no ownership proof", want: false},
		{name: "named mutex owns process state", process: true, want: true},
		{name: "MCP listener provides fallback ownership", mcp: true, want: true},
		{name: "both ownership proofs", process: true, mcp: true, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanRecoverInterruptedTasks(tc.process, tc.mcp); got != tc.want {
				t.Fatalf("CanRecoverInterruptedTasks(%t, %t)=%t want %t", tc.process, tc.mcp, got, tc.want)
			}
		})
	}
}
