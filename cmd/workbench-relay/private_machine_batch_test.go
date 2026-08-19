package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateMachineBatchRejectsProjectEmptyOversizedAndUnknownFields(t *testing.T) {
	base := privateControlEnvelope{
		Version: 1,
		ID:      "batch-12345678",
		Action:  "inspect_machine_batch",
	}

	project := base
	project.Project = "workbench"
	project.Args = json.RawMessage(`{"commands":[{"program":"hostname"}]}`)
	if _, err := executePrivateSafeHands(context.Background(), project, "", ""); err == nil || !strings.Contains(err.Error(), "does not accept a project") {
		t.Fatalf("project-scoped batch should be rejected, got %v", err)
	}

	empty := base
	empty.Args = json.RawMessage(`{"commands":[]}`)
	if _, err := executePrivateSafeHands(context.Background(), empty, "", ""); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("empty batch should be rejected, got %v", err)
	}

	oversized := base
	oversized.Args = json.RawMessage(`{"commands":[{"program":"hostname"},{"program":"hostname"},{"program":"hostname"},{"program":"hostname"},{"program":"hostname"},{"program":"hostname"},{"program":"hostname"},{"program":"hostname"},{"program":"hostname"}]}`)
	if _, err := executePrivateSafeHands(context.Background(), oversized, "", ""); err == nil || !strings.Contains(err.Error(), "at most 8") {
		t.Fatalf("oversized batch should be rejected, got %v", err)
	}

	unknown := base
	unknown.Args = json.RawMessage(`{"commands":[{"program":"hostname","shell":"bash -c whoami"}]}`)
	if _, err := executePrivateSafeHands(context.Background(), unknown, "", ""); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown nested batch field should be rejected, got %v", err)
	}
}

func TestPrivateMachineBatchContinuesAfterMutationPolicyFailure(t *testing.T) {
	env := privateControlEnvelope{
		Version: 1,
		ID:      "batch-continue-1234",
		Action:  "inspect_machine_batch",
		Args: json.RawMessage(`{"commands":[
			{"program":"hostname","timeout_seconds":10},
			{"program":"kubectl","args":["delete","pod","definitely-not-real"],"timeout_seconds":10},
			{"program":"hostname","timeout_seconds":10}
		]}`),
	}
	result, err := executePrivateSafeHands(context.Background(), env, "", "")
	if err != nil {
		t.Fatal(err)
	}
	batch, ok := result["batch"]
	if !ok {
		t.Fatalf("batch result missing: %#v", result)
	}
	encoded, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Commands []struct {
			Index  int    `json:"index"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"commands"`
		OKCount     int `json:"ok_count"`
		FailedCount int `json:"failed_count"`
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got.OKCount != 2 || got.FailedCount != 1 || len(got.Commands) != 3 {
		t.Fatalf("unexpected batch counts: %+v", got)
	}
	if got.Commands[0].Index != 0 || got.Commands[0].Status != "completed" || got.Commands[1].Index != 1 || got.Commands[1].Status != "failed" || got.Commands[2].Index != 2 || got.Commands[2].Status != "completed" {
		t.Fatalf("batch order/isolation broken: %+v", got.Commands)
	}
	if strings.TrimSpace(got.Commands[1].Error) == "" {
		t.Fatalf("mutation-shaped read failure should carry policy error: %+v", got.Commands[1])
	}
}

func TestPrivateControlDoesNotAdvertiseMutationBatchAction(t *testing.T) {
	raw := []byte(`{"version":1,"id":"mutation-batch-1234","action":"run_machine_command_batch","args":{"commands":[]}}`)
	if _, err := decodePrivateControl(raw, "mutation-batch-1234"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("mutation batch must remain unsupported, got %v", err)
	}
}
