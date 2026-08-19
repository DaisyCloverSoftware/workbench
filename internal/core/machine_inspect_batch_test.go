package core

import (
	"context"
	"strings"
	"testing"
)

func TestInspectMachineBatchRejectsEmptyAndOversizedRequests(t *testing.T) {
	if _, err := InspectMachineBatch(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("empty batch should be rejected, got %v", err)
	}

	requests := make([]MachineCommandRequest, MaxMachineInspectBatch+1)
	for i := range requests {
		requests[i] = MachineCommandRequest{Program: "hostname"}
	}
	if _, err := InspectMachineBatch(context.Background(), requests); err == nil || !strings.Contains(err.Error(), "at most 8") {
		t.Fatalf("oversized batch should be rejected, got %v", err)
	}
}

func TestInspectMachineBatchKeepsOrderAndContinuesAfterPolicyFailure(t *testing.T) {
	batch, err := InspectMachineBatch(context.Background(), []MachineCommandRequest{
		{Program: "hostname", TimeoutSeconds: 10},
		{Program: "kubectl", Args: []string{"delete", "pod", "definitely-not-real"}, TimeoutSeconds: 10},
		{Program: "hostname", TimeoutSeconds: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Commands) != 3 || batch.OKCount != 2 || batch.FailedCount != 1 {
		t.Fatalf("unexpected batch counts: %+v", batch)
	}
	for i, item := range batch.Commands {
		if item.Index != i {
			t.Fatalf("item %d has index %d", i, item.Index)
		}
	}
	if batch.Commands[0].Status != "completed" || batch.Commands[2].Status != "completed" {
		t.Fatalf("safe reads should complete around failed item: %+v", batch.Commands)
	}
	if batch.Commands[1].Status != "failed" || strings.TrimSpace(batch.Commands[1].Error) == "" {
		t.Fatalf("mutation-like read should fail policy: %+v", batch.Commands[1])
	}
	if !batch.Commands[0].Command.ReadOnly || !batch.Commands[2].Command.ReadOnly {
		t.Fatalf("successful batch items must remain read-only: %+v", batch.Commands)
	}
}
