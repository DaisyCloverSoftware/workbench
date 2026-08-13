package core

import "testing"

func TestIsWorkerSetupAttention(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "claude denied tools",
			in:   "Both Bash and Write tool calls are denied under the current permission mode—please grant permission (or adjust settings) to allow file creation.",
			want: true,
		},
		{
			name: "generic edit permission",
			in:   "The Edit tool is not allowed to use under the current permission settings.",
			want: true,
		},
		{
			name: "interactive command approval from dogfood",
			in:   "Running go test ./... and any other go/toolchain command in this session requires interactive approval that isn't available in this unattended relay.",
			want: true,
		},
		{
			name: "real production approval",
			in:   "May I restart the production service now?",
			want: false,
		},
		{
			name: "real destructive choice",
			in:   "The migration requires deleting the old table. Approve deletion?",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWorkerSetupAttention(tt.in); got != tt.want {
				t.Fatalf("isWorkerSetupAttention(%q)=%v want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestClassifyWorkerUnavailableSentinel(t *testing.T) {
	got := classifyRunOutput("work attempted\nWORKER_UNAVAILABLE: local Bash policy denied go test")
	if got.WorkerUnavailable == "" || !got.Retryable {
		t.Fatalf("expected retryable worker-unavailable result: %#v", got)
	}
	if got.Attention != "" {
		t.Fatalf("worker-local limitation must not become human attention: %#v", got)
	}
}
