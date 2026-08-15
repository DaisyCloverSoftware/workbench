package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildHarnessJobUsesLeastAuthorityContract(t *testing.T) {
	job := BuildHarnessJob(Task{ID: "task-123", ProjectPath: "/isolated/worktree", Intent: "Implement the feature"}, "worker prompt")
	if job.Version != HarnessProtocolVersion || job.TaskID != "task-123" {
		t.Fatalf("unexpected job identity: %#v", job)
	}
	if !job.Capabilities.RepositoryRead || !job.Capabilities.RepositoryWrite || !job.Capabilities.LocalCommands {
		t.Fatalf("expected isolated repository capabilities: %#v", job.Capabilities)
	}
	if job.Capabilities.NetworkAccess || job.Capabilities.Publish || job.Capabilities.Deploy || job.Capabilities.Secrets {
		t.Fatalf("structured harness was granted remote/secret authority: %#v", job.Capabilities)
	}
	body, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"remote_url", "publication_target", "credentials", "vault", "provider_account"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("structured job exposed forbidden field %q: %s", forbidden, body)
		}
	}
}

func TestDecodeHarnessJobResultIsStrictAndIdentityNeutral(t *testing.T) {
	result, err := decodeHarnessJobResult([]byte(`{"version":1,"task_id":"task-1","status":"completed","report":"done"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "task-1" || result.Status != HarnessJobCompleted || result.Report != "done" {
		t.Fatalf("unexpected decoded result: %#v", result)
	}
	if _, err := decodeHarnessJobResult([]byte(`{"version":1,"task_id":"task-1","status":"completed","unknown":true}`)); err == nil {
		t.Fatal("unknown structured harness fields were accepted")
	}
	if _, err := decodeHarnessJobResult([]byte(`{"version":2,"task_id":"task-1","status":"completed"}`)); err == nil {
		t.Fatal("unsupported protocol version was accepted")
	}
	if _, err := decodeHarnessJobResult([]byte("{\"version\":1,\"task_id\":\"task-1\",\"status\":\"completed\"}\n{}")); err == nil {
		t.Fatal("multiple JSON result values were accepted")
	}
}

func TestHarnessResultMapsExplicitStatusWithoutParsingProse(t *testing.T) {
	attention, err := harnessResultToRunResult(HarnessJobResult{
		Version: HarnessProtocolVersion,
		TaskID:  "task-1",
		Status:  HarnessJobNeedsAttention,
		Report:  "partial report without marker text",
		Attention: "Choose the public API shape.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if attention.Attention != "Choose the public API shape." || attention.Output != "partial report without marker text" {
		t.Fatalf("attention result was not mapped structurally: %#v", attention)
	}

	unavailable, err := harnessResultToRunResult(HarnessJobResult{
		Version:     HarnessProtocolVersion,
		TaskID:      "task-1",
		Status:      HarnessJobUnavailable,
		Unavailable: "sign-in is required",
		Category:    HarnessFailureAuthentication,
	})
	if err == nil {
		t.Fatal("unavailable harness should return a routing error")
	}
	if !unavailable.Retryable || !unavailable.Authentication || unavailable.WorkerUnavailable != "sign-in is required" {
		t.Fatalf("unavailable result lost routing metadata: %#v", unavailable)
	}
}

func TestHarnessControlFieldsAreBounded(t *testing.T) {
	long := strings.Repeat("x", maxWorkerControlTextBytes*2)
	result, err := decodeHarnessJobResult([]byte(`{"version":1,"task_id":"task-1","status":"needs_attention","attention":"` + long + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Attention) > maxWorkerControlTextBytes+64 {
		t.Fatalf("attention field remained unbounded: %d bytes", len(result.Attention))
	}
}
