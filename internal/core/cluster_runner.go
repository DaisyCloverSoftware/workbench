package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunnerRequest is the structured task envelope sent from a Workbench desktop
// to a Workbench Runner on another machine. The runner re-routes locally so the
// desktop does not need to know which coding CLIs are installed on that host.
type RunnerRequest struct {
	Task            Task `json:"task"`
	AvoidWorkUsage  bool `json:"avoid_work_usage"`
	AllowMeteredAPI bool `json:"allow_metered_api"`
}

// RunnerResponse is deliberately small and transport-neutral. A future HTTP or
// message-queue transport can use exactly the same envelope as the SSH adapter.
type RunnerResponse struct {
	Result       RunResult `json:"result"`
	ProviderID   string    `json:"provider_id,omitempty"`
	ProviderName string    `json:"provider_name,omitempty"`
	ProviderCost CostClass `json:"provider_cost,omitempty"`
	Error        string    `json:"error,omitempty"`
	Attempts     []string  `json:"attempts,omitempty"`
}

// RunClusterRunnerSSH sends a task to the cluster-side Workbench Runner using
// stdin rather than shell arguments. That avoids prompt quoting bugs and keeps
// the SSH transport useful even for very large worker instructions.
func RunClusterRunnerSSH(ctx context.Context, host string, task Task, prefs Preferences) (RunResult, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return RunResult{Retryable: true}, errors.New("cluster runner SSH host is not configured")
	}
	reqBody, err := json.Marshal(RunnerRequest{
		Task:            task,
		AvoidWorkUsage:  prefs.AvoidWorkUsage,
		AllowMeteredAPI: prefs.AllowMeteredAPI,
	})
	if err != nil {
		return RunResult{}, err
	}

	cmd := exec.CommandContext(ctx,
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
		"$HOME/.local/bin/workbench-runner", "run",
	)
	configureChildProcess(cmd, false)
	cmd.Stdin = bytes.NewReader(reqBody)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	var rr RunnerResponse
	decodeErr := json.Unmarshal(stdout.Bytes(), &rr)
	if decodeErr == nil {
		res := rr.Result
		res.WorkerProviderID = rr.ProviderID
		res.WorkerProviderName = rr.ProviderName
		res.WorkerCost = rr.ProviderCost
		if strings.TrimSpace(rr.Error) != "" {
			res.Retryable = true
			return res, errors.New(rr.Error)
		}
		if runErr != nil {
			res.Retryable = true
			return res, runErr
		}
		return res, nil
	}

	combined := strings.TrimSpace(stdout.String())
	if se := strings.TrimSpace(stderr.String()); se != "" {
		if combined != "" {
			combined += "\n\n"
		}
		combined += se
	}
	res := classifyRunOutput(combined)
	res.Retryable = true
	if runErr != nil {
		low := strings.ToLower(combined + " " + runErr.Error())
		res.Authentication = strings.Contains(low, "permission denied") || strings.Contains(low, "publickey") || strings.Contains(low, "authentication")
		if combined == "" {
			res.Output = runErr.Error()
		}
		return res, fmt.Errorf("cluster runner SSH failed: %w", runErr)
	}
	return res, fmt.Errorf("cluster runner returned invalid response: %w", decodeErr)
}

// ExecuteRunnerRequest is the cluster-side routing loop. It uses the same cost
// policy as the desktop and therefore prefers zero-marginal/included workers
// before scarce Work/Codex, while leaving metered APIs disabled unless opted in.
func ExecuteRunnerRequest(ctx context.Context, req RunnerRequest) RunnerResponse {
	task := req.Task
	resolved, err := resolveRunnerProject(task.ProjectPath)
	if err != nil {
		return RunnerResponse{Error: err.Error()}
	}
	task.ProjectPath = resolved
	prefs := Preferences{AvoidWorkUsage: req.AvoidWorkUsage, AllowMeteredAPI: req.AllowMeteredAPI}
	providers := ScanProviders()
	candidates := routeCandidates(providers, prefs, task)

	var attempts []string
	for _, p := range candidates {
		// A runner must never recursively delegate to another remote Workbench
		// runner through the same SSH configuration.
		if p.ID == "workbench-runner" {
			continue
		}
		res, runErr := RunProvider(ctx, p, task, prefs)
		attempts = append(attempts, fmt.Sprintf("%s: %s", p.Name, attemptSummary(res, runErr)))
		if strings.TrimSpace(res.Attention) != "" {
			return RunnerResponse{Result: res, ProviderID: p.ID, ProviderName: p.Name, ProviderCost: p.Cost, Attempts: attempts}
		}
		if runErr == nil {
			return RunnerResponse{Result: res, ProviderID: p.ID, ProviderName: p.Name, ProviderCost: p.Cost, Attempts: attempts}
		}
	}
	if len(candidates) == 0 {
		return RunnerResponse{Error: "no eligible coding worker is installed on the runner host"}
	}
	return RunnerResponse{Error: "every eligible runner worker failed", Attempts: attempts}
}

func runnerRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_ROOT")); root != "" {
		return filepath.Abs(root)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "src"), nil
}

func resolveRunnerProject(requested string) (string, error) {
	root, err := runnerRoot()
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return "", fmt.Errorf("runner root is not a directory: %s", root)
	}

	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", errors.New("project path is empty")
	}

	// First accept a real path on the runner host, but only if it stays inside
	// the configured root.
	if abs, absErr := filepath.Abs(requested); absErr == nil {
		if st, statErr := os.Stat(abs); statErr == nil && st.IsDir() {
			if withinRoot(root, abs) {
				return abs, nil
			}
			return "", fmt.Errorf("project is outside WORKBENCH_RUNNER_ROOT: %s", abs)
		}
	}

	// Desktop paths are often Windows paths. Map their final directory name to
	// ~/src/<repo-name> on the runner. This makes a cloned repository portable
	// across Windows desktop + Linux cluster without hand-editing every task.
	normalized := strings.ReplaceAll(requested, "\\", "/")
	name := filepath.Base(strings.TrimRight(normalized, "/"))
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("cannot derive repository name from %q", requested)
	}
	candidate := filepath.Join(root, name)
	st, statErr := os.Stat(candidate)
	if statErr != nil || !st.IsDir() {
		return "", fmt.Errorf("runner cannot find project %q; expected %s", requested, candidate)
	}
	return candidate, nil
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
