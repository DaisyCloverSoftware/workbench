package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func operationStatusRepo(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixed process fixture")
	}
	repo := newOperationsScriptTestRepo(t)
	name := filepath.Join(repo, "scripts", "ops", "status-fixture.sh")
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("#!/bin/bash\nset -eu\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "scripts/ops/status-fixture.sh")
	gitTest(t, repo, "commit", "-m", "fixed isolated status fixture")
	return repo
}
func TestOperationsStatusRetainsFactsOnWithholdingAndFailure(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		exit        int
		withheld    bool
	}{
		{"success", "", 0, false},
		{"nonzero", "exit 7", 7, false},
		{"protected_stdout", "printf '%s\\n' 'OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz'", 0, true},
		{"protected_stderr_nonzero", "printf '%s\\n' 'OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz' >&2; exit 9", 9, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := operationStatusRepo(t, "printf '%s' '"+testDisposition+"' > \"$WORKBENCH_OPERATION_STATUS_FILE\"\n"+tc.extra)
			r, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/status-fixture.sh"})
			if (err != nil) != (tc.exit != 0 || tc.withheld) {
				t.Fatal("unexpected error state")
			}
			s := r.Status
			if s == nil || !s.Started || s.Completion != "exited" || s.ProcessExitCode == nil || *s.ProcessExitCode != tc.exit || !s.ProcessObserved {
				t.Fatalf("lost process facts: %+v", s)
			}
			if s.OutputWithheld != tc.withheld || s.Reporting != "complete" || s.Report == nil || s.Report.Update != "confirmed" {
				t.Fatal("lost protected status")
			}
			if tc.withheld && (strings.Contains(r.Output, "sk-example") || s.EvidenceComplete) {
				t.Fatal("protected output escaped or evidence falsely complete")
			}
		})
	}
}
func TestOperationsStatusTimeoutAndInterruption(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		timeout          int
	}{
		{"timeout", "sleep 20", "timed_out", 1},
		{"interrupted", "kill -KILL $$", "interrupted", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := operationStatusRepo(t, tc.body)
			begin := time.Now()
			r, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/status-fixture.sh", TimeoutSeconds: tc.timeout})
			if err == nil || r.Status == nil || !r.Status.Started || r.Status.Completion != tc.want || r.Status.EvidenceComplete {
				t.Fatal("incorrect interruption evidence")
			}
			if r.Status.ProcessExitCode != nil {
				t.Fatal("fabricated normal exit code")
			}
			if time.Since(begin) > 10*time.Second {
				t.Fatal("unbounded process wait")
			}
		})
	}
}
func TestOperationsStatusNoStartAndMissingReporting(t *testing.T) {
	r, err := RunOperationsScript(context.Background(), t.TempDir(), OperationsScriptRequest{Path: "not-allowed.sh"})
	if err == nil || r.Status == nil || r.Status.Started || r.Status.Completion != "not_started" || r.Status.ProcessExitCode != nil {
		t.Fatal("fabricated execution")
	}
	repo := operationStatusRepo(t, "echo harmless")
	r, err = RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/status-fixture.sh"})
	if err != nil || r.Status.Reporting != "not_reported" || r.Status.Report != nil || r.Status.EvidenceComplete {
		t.Fatal("missing application evidence invented")
	}
}
func TestOperationsStatusReportingFailureDoesNotRewriteProcessResult(t *testing.T) {
	repo := operationStatusRepo(t, "printf '{' > \"$WORKBENCH_OPERATION_STATUS_FILE\"")
	r, err := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/status-fixture.sh"})
	if err == nil || r.Status.Reporting != "invalid" || r.Status.Report != nil || r.Status.ProcessExitCode == nil || *r.Status.ProcessExitCode != 0 || r.Status.EvidenceComplete {
		t.Fatal("report failure confused with process failure")
	}
}

func TestOperationsStatusDoesNotDeriveReportFromOutput(t *testing.T) {
	repo := operationStatusRepo(t, "printf '%s\\n' '"+testDisposition+"'")
	r, _ := RunOperationsScript(context.Background(), repo, OperationsScriptRequest{Path: "scripts/ops/status-fixture.sh"})
	if r.Status.Report != nil || r.Status.Reporting != "not_reported" {
		t.Fatal("arbitrary output became a status record")
	}
}
