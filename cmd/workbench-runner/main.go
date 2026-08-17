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

const runnerVersion = "0.9.4"
const runnerUsage = "usage: workbench-runner <run|job <submit|status|cancel>|tool-json|provider-login <provider-id>|cloud-models|cloud-model-set <model>|inspect <project-directory>|snapshot <project-directory>|prepare <project-directory> <task-id>|policy <get|prepare|publish|delete> <project-directory> [remote-url]|harness <get|set|delete> [adapter-executable]|update <check|apply>|doctor|selftest|live-selftest|version>"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		run()
	case "job":
		job()
	case "job-execute":
		jobExecute()
	case "review-json":
		reviewJSON()
	case "tool-json":
		toolJSON()
	case "provider-login":
		providerLogin()
	case "cloud-models":
		cloudModels()
	case "cloud-model-set":
		cloudModelSet()
	case "agent":
		// Internal OpenClaw shim. The runner provider overlay invokes this with
		// the existing fixed Workbench OpenClaw argument shape; it is not a
		// model-safe command or an arbitrary OpenClaw proxy.
		openClawAgent()
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
	case "harness":
		harness()
	case "update":
		update()
	case "doctor":
		doctor()
	case "selftest":
		if err := selftest(); err != nil {
			fmt.Fprintln(os.Stderr, "SELFTEST FAILED:", err)
			os.Exit(1)
		}
	case "live-selftest":
		if err := liveSelftest(); err != nil {
			fmt.Fprintln(os.Stderr, "LIVE SELFTEST FAILED:", err)
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

func providerLogin() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: workbench-runner provider-login <provider-id>")
		os.Exit(2)
	}
	if err := core.RunProviderLoginInteractive(os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "provider login:", err)
		os.Exit(1)
	}
}

func cloudModels() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: workbench-runner cloud-models")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := core.DiscoverOpenClawCloudModels(ctx, "")
	if err != nil {
		write(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	write(map[string]any{"ok": true, "catalog": catalog})
}

func cloudModelSet() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: workbench-runner cloud-model-set <model>")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	catalog, err := core.SetOpenClawCloudDefault(ctx, "", os.Args[2])
	if err != nil {
		write(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}
	write(map[string]any{"ok": true, "default_model": catalog.DefaultModel})
}

func openClawAgent() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	code := core.RunOpenClawCloudAgentCLIWithTaskOverride(ctx, os.Args[2:], os.Stdout, os.Stderr)
	if code != 0 {
		os.Exit(code)
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
	providers := core.ApplyProviderHealth(core.ScanProviders(), time.Now())
	harnessStatus := core.RunnerHarnessConfigurationStatus()
	fmt.Printf("Workbench Runner %s\n", runnerVersion)
	fmt.Println("Provider scan:")
	for _, p := range providers {
		marker := "-"
		if p.Installed {
			marker = "+"
		}
		fmt.Printf("  %s %-22s %-22s %s\n", marker, p.Name, p.Cost, p.Status)
	}
	fmt.Printf("Structured harness: %s", harnessStatus.Status)
	if harnessStatus.AdapterName != "" {
		fmt.Printf(" (%s)", harnessStatus.AdapterName)
	}
	fmt.Println()
	fmt.Println("\nRunner root: set WORKBENCH_RUNNER_ROOT to override; default is ~/src")
	fmt.Println("Provider cooldowns: retryable provider/setup failures are skipped briefly; Rescan in the native app clears local cooldowns after fixing setup")
	fmt.Println("Durable remote work: workbench-runner job <submit|status|cancel>; jobs survive the submitting SSH session")
	fmt.Println("Bounded desktop/Chat tools: workbench-runner tool-json")
	fmt.Println("Human provider authentication: workbench-runner provider-login <provider-id>")
	fmt.Println("Cloud model inventory (read-only): workbench-runner cloud-models")
	fmt.Println("Cloud default (operator-only): workbench-runner cloud-model-set <model>")
	fmt.Println("Changeset inspection: workbench-runner inspect <project-directory>")
	fmt.Println("Stable changeset snapshot: workbench-runner snapshot <project-directory>")
	fmt.Println("Isolated local preparation: workbench-runner prepare <project-directory> <task-id>")
	fmt.Println("Publication policy (operator-only): workbench-runner policy <get|prepare|publish|delete> ...")
	fmt.Println("Host harness configuration (operator-only): workbench-runner harness <get|set|delete> ...")
	fmt.Println("Local maintenance (operator-only): workbench-runner update <check|apply>")
	fmt.Println("Deterministic Workbench health proof: workbench-runner selftest")
	fmt.Println("External AI worker availability proof: workbench-runner live-selftest")
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
