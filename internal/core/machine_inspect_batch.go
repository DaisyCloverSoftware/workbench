package core

import (
	"context"
	"errors"
	"strings"
)

const MaxMachineInspectBatch = 8

type MachineInspectBatchItem struct {
	Index   int                  `json:"index"`
	Status  string               `json:"status"`
	Command MachineCommandResult `json:"command"`
	Error   string               `json:"error,omitempty"`
}

type MachineInspectBatchResult struct {
	Commands    []MachineInspectBatchItem `json:"commands"`
	OKCount     int                       `json:"ok_count"`
	FailedCount int                       `json:"failed_count"`
}

// InspectMachineBatch runs a small ordered set of read-only machine inspections.
// Every item goes through InspectMachine independently, so this helper cannot
// broaden the existing machine-command policy. Item failures are isolated and
// later reads still execute; mutations remain intentionally single-command only.
func InspectMachineBatch(ctx context.Context, requests []MachineCommandRequest) (MachineInspectBatchResult, error) {
	if len(requests) == 0 {
		return MachineInspectBatchResult{}, errors.New("machine inspection batch requires at least one command")
	}
	if len(requests) > MaxMachineInspectBatch {
		return MachineInspectBatchResult{}, errors.New("machine inspection batch supports at most 8 commands")
	}

	batch := MachineInspectBatchResult{Commands: make([]MachineInspectBatchItem, 0, len(requests))}
	for i, request := range requests {
		result, err := InspectMachine(ctx, request)
		item := MachineInspectBatchItem{
			Index:   i,
			Status:  "completed",
			Command: result,
		}
		if err != nil {
			item.Status = "failed"
			item.Error = strings.TrimSpace(err.Error())
			batch.FailedCount++
		} else {
			batch.OKCount++
		}
		batch.Commands = append(batch.Commands, item)
	}
	return batch, nil
}
