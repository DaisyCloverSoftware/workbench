package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func job() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: workbench-runner job <submit|status|cancel> [task-id]")
		os.Exit(2)
	}
	switch os.Args[2] {
	case "submit":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: workbench-runner job submit")
			os.Exit(2)
		}
		dec := json.NewDecoder(os.Stdin)
		dec.DisallowUnknownFields()
		var req core.RunnerRequest
		if err := dec.Decode(&req); err != nil {
			write(map[string]any{"ok": false, "error": "invalid runner request: " + err.Error()})
			os.Exit(2)
		}
		result, err := core.SubmitRunnerJob(req)
		if err != nil {
			write(map[string]any{"ok": false, "error": err.Error()})
			os.Exit(1)
		}
		write(map[string]any{"ok": true, "result": result})
	case "status":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: workbench-runner job status <task-id>")
			os.Exit(2)
		}
		result, err := core.GetRunnerJob(os.Args[3])
		if err != nil {
			write(map[string]any{"ok": false, "error": err.Error()})
			os.Exit(1)
		}
		write(map[string]any{"ok": true, "job": result})
	case "cancel":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: workbench-runner job cancel <task-id>")
			os.Exit(2)
		}
		result, err := core.CancelRunnerJob(os.Args[3])
		if err != nil {
			write(map[string]any{"ok": false, "error": err.Error()})
			os.Exit(1)
		}
		write(map[string]any{"ok": true, "job": result})
	default:
		fmt.Fprintln(os.Stderr, "usage: workbench-runner job <submit|status|cancel> [task-id]")
		os.Exit(2)
	}
}

func jobExecute() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "job-execute is an internal Workbench runner command")
		os.Exit(2)
	}
	if err := core.ExecuteStoredRunnerJob(os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "workbench-runner job-execute:", err)
		os.Exit(1)
	}
}
