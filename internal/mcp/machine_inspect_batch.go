package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func machineInspectBatchTool(machineProgram, machineArgs, machineTimeout map[string]any) map[string]any {
	command := objSchema(map[string]any{
		"program":         machineProgram,
		"args":            machineArgs,
		"timeout_seconds": machineTimeout,
	}, []string{"program"})
	commands := map[string]any{
		"type":        "array",
		"description": "One to eight ordered read-only machine inspections. Each item is independently validated by the same policy as inspect_machine and executes sequentially.",
		"minItems":    1,
		"maxItems":    core.MaxMachineInspectBatch,
		"items":       command,
	}
	return tool(
		"inspect_machine_batch",
		"Inspect Workbench machine batch",
		"Run 1-8 ordered read-only allowlisted host/cluster commands through the exact inspect_machine policy. Commands execute sequentially; one rejected or failed item does not stop later reads. There is deliberately no batched mutation equivalent.",
		objSchema(map[string]any{"commands": commands}, []string{"commands"}),
		anyObjectSchema(),
		annotations(true, false, true),
	)
}

func machineInspectBatchRequestsArg(a map[string]any) ([]core.MachineCommandRequest, error) {
	raw, ok := a["commands"]
	if !ok {
		return nil, errors.New("inspect_machine_batch commands are required")
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, errors.New("inspect_machine_batch commands must be an array")
	}
	if len(values) == 0 {
		return nil, errors.New("machine inspection batch requires at least one command")
	}
	if len(values) > core.MaxMachineInspectBatch {
		return nil, errors.New("machine inspection batch supports at most 8 commands")
	}

	requests := make([]core.MachineCommandRequest, 0, len(values))
	for i, rawCommand := range values {
		command, ok := rawCommand.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("inspect_machine_batch command %d must be an object", i)
		}
		for key := range command {
			switch key {
			case "program", "args", "timeout_seconds":
			default:
				return nil, fmt.Errorf("inspect_machine_batch command %d contains unknown field %q", i, key)
			}
		}
		program, ok := command["program"].(string)
		program = strings.TrimSpace(program)
		if !ok || program == "" {
			return nil, fmt.Errorf("inspect_machine_batch command %d program is required", i)
		}
		args, err := strictRawStringArray(command, "args")
		if err != nil {
			return nil, fmt.Errorf("inspect_machine_batch command %d: %w", i, err)
		}
		timeout, err := strictOptionalInt(command, "timeout_seconds")
		if err != nil {
			return nil, fmt.Errorf("inspect_machine_batch command %d: %w", i, err)
		}
		requests = append(requests, core.MachineCommandRequest{Program: program, Args: args, TimeoutSeconds: timeout})
	}
	return requests, nil
}

func strictRawStringArray(a map[string]any, key string) ([]string, error) {
	raw, ok := a[key]
	if !ok {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s must contain only strings", key)
		}
		out = append(out, s)
	}
	return out, nil
}

func strictOptionalInt(a map[string]any, key string) (int, error) {
	raw, ok := a[key]
	if !ok {
		return 0, nil
	}
	switch value := raw.(type) {
	case float64:
		if value != float64(int(value)) {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return int(value), nil
	case int:
		return value, nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}

func callMachineInspectBatch(ctx context.Context, a map[string]any) any {
	requests, err := machineInspectBatchRequestsArg(a)
	if err != nil {
		return textContent(map[string]any{"error": err.Error()}, true)
	}
	batch, err := core.InspectMachineBatch(ctx, requests)
	if err != nil {
		return textContent(map[string]any{"error": err.Error()}, true)
	}
	return textContent(map[string]any{"ok": batch.FailedCount == 0, "batch": batch}, false)
}
