package core

import (
	"errors"
	"testing"
)

func TestOperationModelCapacityFailureAllowsCapacityWorkerUnavailable(t *testing.T) {
	for _, workerUnavailable := range []string{
		"Context overflow: prompt too large for the model.",
		"You've reached your Codex subscription usage limit.",
	} {
		res := RunResult{
			Output:            workerUnavailable,
			WorkerUnavailable: workerUnavailable,
			Retryable:         true,
		}
		if !operationModelCapacityFailure(res, errors.New("OpenClaw is unavailable for this operational task")) {
			t.Fatalf("capacity worker-unavailable should remain eligible for model fallback: %q", workerUnavailable)
		}
		if operationFallbackMustStop(res) {
			t.Fatalf("capacity worker-unavailable should not stop the remaining model fallback chain: %q", workerUnavailable)
		}
	}
}

func TestOperationModelCapacityFailureKeepsUnrelatedWorkerUnavailableAuthoritative(t *testing.T) {
	res := RunResult{WorkerUnavailable: "kubectl is missing", Retryable: true}
	if operationModelCapacityFailure(res, errors.New("quota text from an outer wrapper")) {
		t.Fatal("unrelated worker-local unavailability must not be reclassified from an outer capacity-looking error")
	}
	if !operationFallbackMustStop(res) {
		t.Fatal("unrelated worker-local unavailability should stop model fallback")
	}
}
