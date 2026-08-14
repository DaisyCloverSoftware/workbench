package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const runnerVersion = "0.5.0"
const runnerUsage = "usage: workbench-runner <run|inspect <project-directory>|snapshot <project-directory>|prepare <project-directory> <task-id>|policy <get|prepare|publish|delete> <project-directory> [remote-url]|doctor|selftest|version>"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		run()
	case "inspect":
		inspect()
	case "snapshot":
		snapshot()
	case "prepare":
		prepare()
	case "policy":
		policy()
	case "policy-json":
		policyJSON()
	case "doctor":
		doctor()
	case "selftest":
		if err := selftest(); err != nil {
			fmt.Fprintln(os.Stderr, "SELFTEST FAILED:", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println(runnerVersion)
	default:
		usage()
		os.Exit(2)
	}
}

func run() {
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	var req core.RunnerRequest
	if err := dec.Decode(&req); err != nil {
		write(core.RunnerResponse{Error: "invalid runner request: " + err.Error()})
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	resp := core.ExecuteRunnerRequest(ctx, req)
	write(resp)
	if strings.TrimSpace(resp.Error) != "" {
		os.Exit(1)
	}
}

func inspect() {
	if len(os.Args) != 3 {
		usage()
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := core.InspectChangeset(ctx, os.Args[2])
	if err != nil {
		write(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	write(map[string]any{"ok": true, "changeset": result})
}

func snapshot() {
	if len(os.Args) != 3 {
		usage()
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := core.SnapshotChangeset(ctx, os.Args[2])
	if err != nil {
		write(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	write(map[string]any{"ok": true, "snapshot": result})
}

func prepare() {
	if len(os.Args) != 4 {
		usage()
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := core.PrepareChangeset(ctx, os.Args[2], os.Args[3])
	if err != nil {
		write(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	write(map[string]any{"ok": true, "prepared": result})
}

func policy() {
	response, err := applyPublicationPolicyCommand(os.Args[2:])
	if err != nil {
		write(core.RunnerPolicyResponse{OK: false, Error: err.Error()})
		os.Exit(1)
	}
	write(response)
}

func policyJSON() {
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	var req core.RunnerPolicyRequest
	if err := dec.Decode(&req); err != nil {
		write(core.RunnerPolicyResponse{OK: false, Error: "invalid runner policy request: " + err.Error()})
		os.Exit(2)
	}
	response, err := core.ApplyRunnerPublicationPolicy(req)
	if err != nil {
		write(core.RunnerPolicyResponse{OK: false, Error: err.Error()})
		os.Exit(1)
	}
	write(response)
}

func applyPublicationPolicyCommand(args []string) (core.RunnerPolicyResponse, error) {
	if len(args) < 2 {
		return core.RunnerPolicyResponse{}, errors.New("publication policy requires an action and project directory")
	}
	action := strings.ToLower(strings.TrimSpace(args[0]))
	var expected int
	switch action {
	case "get", "prepare", "delete":
		expected = 2
	case "publish":
		expected = 3
	default:
		return core.RunnerPolicyResponse{}, errors.New("publication policy action must be get, prepare, publish or delete")
	}
	if len(args) != expected {
		if action == "publish" {
			return core.RunnerPolicyResponse{}, errors.New("usage: workbench-runner policy publish <project-directory> <remote-url>")
		}
		return core.RunnerPolicyResponse{}, fmt.Errorf("usage: workbench-runner policy %s <project-directory>", action)
	}
	req := core.RunnerPolicyRequest{Action: action, Project: args[1]}
	if action == "publish" {
		req.RemoteURL = args[2]
	}
	return core.ApplyRunnerPublicationPolicy(req)
}

func doctor() {
	providers := core.ScanProviders()
	fmt.Printf("Workbench Runner %s\n", runnerVersion)
	fmt.Println("Provider scan:")
	for _, p := range providers {
		marker := "-"
		if p.Installed {
			marker = "+"
		}
		fmt.Printf("  %s %-22s %-22s %s\n", marker, p.Name, p.Cost, p.Status)
	}
	fmt.Println("\nRunner root: set WORKBENCH_RUNNER_ROOT to override; default is ~/src")
	fmt.Println("Changeset inspection: workbench-runner inspect <project-directory>")
	fmt.Println("Stable changeset snapshot: workbench-runner snapshot <project-directory>")
	fmt.Println("Isolated local preparation: workbench-runner prepare <project-directory> <task-id>")
	fmt.Println("Publication policy (operator-only): workbench-runner policy <get|prepare|publish|delete> ...")
	fmt.Println("Live proof: workbench-runner selftest")
}

func write(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, runnerUsage)
}
