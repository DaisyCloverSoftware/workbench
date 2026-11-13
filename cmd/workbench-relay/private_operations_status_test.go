package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const relayTestDisposition = `{"version":1,"terminal":true,"stage":"readiness","reason":"readiness_failed","update":"confirmed","recovery":"confirmed","authentication":"passed","health":"200","readiness":"503","audit":"complete"}`

func TestPrivateOperationsStatusSurvivesExecutionErrorAndWithholding(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX isolated process integration")
	}
	for _, tc := range []struct {
		name, tail string
		exit       int
		withheld   bool
	}{
		{"nonzero", "exit 7", 7, false},
		{"withheld_zero", "echo 'OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz'", 0, true},
		{"withheld_nonzero", "echo 'OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz' >&2; exit 9", 9, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "sample")
			if err := os.Mkdir(repo, 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("WORKBENCH_RUNNER_ROOT", root)
			t.Setenv("HOME", t.TempDir())
			gitRelayTest(t, repo, "init")
			gitRelayTest(t, repo, "config", "user.email", "fixture@example.invalid")
			gitRelayTest(t, repo, "config", "user.name", "Isolated fixture")
			path := filepath.Join(repo, "scripts", "ops", "report.sh")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			body := "#!/bin/bash\nset -eu\nprintf '%s' '" + relayTestDisposition + "' > \"$WORKBENCH_OPERATION_STATUS_FILE\"\n" + tc.tail + "\n"
			if err := os.WriteFile(path, []byte(body), 0755); err != nil {
				t.Fatal(err)
			}
			gitRelayTest(t, repo, "add", ".")
			gitRelayTest(t, repo, "commit", "-m", "synthetic result fixture")
			head := strings.TrimSpace(gitRelayTest(t, repo, "rev-parse", "HEAD"))
			env := privateControlEnvelope{Version: 1, ID: "result-status-test", Action: "run_operations_script", Project: "sample", Args: json.RawMessage(`{"path":"scripts/ops/report.sh"}`)}
			out := executePrivateControlCandidate(context.Background(), privateControlCandidate{id: env.ID, env: env}, "", "http://127.0.0.1:1", "unused")
			if out.Status != "failed" || out.OperationStatus == nil {
				t.Fatal("execution failure lost result")
			}
			wire, err := marshalPrivateControlOutbox(out)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(wire), "sk-example-abcdefghijklmnopqrstuvwxyz") {
				t.Fatal("sensitive fixture escaped protection")
			}
			var got privateControlOutbox
			if json.Unmarshal(wire, &got) != nil {
				t.Fatal("invalid outbox")
			}
			s := got.OperationStatus
			if got.ID != env.ID || s == nil || s.SourceCommit != head || len(s.ScriptSHA256) != 64 || !s.Started || s.Completion != "exited" || !s.ProcessObserved || s.ProcessExitCode == nil || *s.ProcessExitCode != tc.exit || s.OutputWithheld != tc.withheld {
				t.Fatal("incorrect executor facts")
			}
			if s.Report == nil || s.Report.Stage != "readiness" || s.Report.Reason != "readiness_failed" || s.Report.Update != "confirmed" || s.Report.Recovery != "confirmed" {
				t.Fatal("closed disposition lost")
			}
		})
	}
}

func TestPrivateOperationsStatusSurvivesOuterPayloadProtection(t *testing.T) {
	code := 0
	status := &core.OperationsStatus{Version: 1, ExecutorVersion: core.Version, Started: true, Completion: "exited", ProcessObserved: true, ProcessExitCode: &code, EvidenceComplete: true, Reporting: "not_reported"}
	out := privateControlOutbox{Version: 1, ID: "fixed-fixture", Status: "completed", Action: "run_operations_script", OperationStatus: status, Result: map[string]any{"output": "OPENAI_API_KEY=sk-example-abcdefghijklmnopqrstuvwxyz"}}
	b, err := marshalPrivateControlOutbox(out)
	if err != nil {
		t.Fatal(err)
	}
	var got privateControlOutbox
	_ = json.Unmarshal(b, &got)
	if got.Result != nil || got.Status != "failed" || got.OperationStatus == nil || !got.OperationStatus.OutputWithheld || got.OperationStatus.EvidenceComplete || !got.OperationStatus.Started || *got.OperationStatus.ProcessExitCode != 0 {
		t.Fatal("payload protection discarded or fabricated execution status")
	}
	if strings.Contains(string(b), "sk-example-abcdefghijklmnopqrstuvwxyz") {
		t.Fatal("protected data leaked")
	}
	if status.OutputWithheld || !status.EvidenceComplete {
		t.Fatal("serialization modified shared original result")
	}
}

func TestPrivateOperationsStatusDoesNotInventHistoricalResults(t *testing.T) {
	out := privateControlOutbox{Version: 1, ID: "old-fixture", Status: "failed", Error: "output withheld"}
	b, err := marshalPrivateControlOutbox(out)
	if err != nil || strings.Contains(string(b), "operation_status") {
		t.Fatal("historical result synthesized")
	}
}
