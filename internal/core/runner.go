package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type RunResult struct {
	Output             string            `json:"output,omitempty"`
	Attention          string            `json:"attention,omitempty"`
	WorkerUnavailable  string            `json:"worker_unavailable,omitempty"`
	Retryable          bool              `json:"retryable,omitempty"`
	Authentication     bool              `json:"authentication,omitempty"`
	WorkerProviderID   string            `json:"worker_provider_id,omitempty"`
	WorkerProviderName string            `json:"worker_provider_name,omitempty"`
	WorkerCost         CostClass         `json:"worker_cost,omitempty"`
	Review             *TaskReviewResult `json:"review,omitempty"`
}

func BuildWorkerPrompt(task Task) string {
	var b strings.Builder
	b.WriteString("You are a coding worker operating under Workbench. Complete the user's intent autonomously.\n\n")
	b.WriteString("Project/workspace:\n")
	b.WriteString(task.ProjectPath)
	b.WriteString("\n\nIntent:\n")
	b.WriteString(task.Intent)
	b.WriteString("\n\nRules:\n")
	b.WriteString("- Work only inside the provided project/repository.\n")
	b.WriteString("- You may inspect files, edit source, and run local build/test/lint commands.\n")
	b.WriteString("- Do NOT deploy, push to remotes, change production, spend money, reveal credentials, or perform destructive operations.\n")
	b.WriteString("- Prefer fixing and retesting over stopping for ordinary implementation choices.\n")
	b.WriteString("- If your own local harness, login, quota, sandbox, command approval, or tool permission prevents requested work, do not ask the human to approve the worker. End with WORKER_UNAVAILABLE: followed by one concise reason so Workbench can route another eligible worker.\n")
	b.WriteString("- If and only if a genuinely human product decision/permission is required, stop and output a final line beginning exactly ATTENTION_REQUIRED: followed by one concise question.\n")
	b.WriteString("- Otherwise finish the work and provide a concise completion report: changes, verification, and any non-blocking warnings.\n")
	b.WriteString("- If successful work reveals genuinely durable project-specific knowledge that should prevent future rediscovery, append up to three final lines using exactly WORKBENCH_MEMORY: followed by one JSON object with kind, title, content, and optional tags. Allowed kinds: fact, decision, constraint, pattern, routine, code. Do not include secrets, raw logs, transient status, account details, machine identifiers, or a scope field; Workbench stores worker memories at project scope only.\n")
	if strings.TrimSpace(task.HumanAnswer) != "" {
		b.WriteString("\nHuman answer to the previous attention request:\n")
		b.WriteString(task.HumanAnswer)
		b.WriteString("\nContinue from that decision; do not ask the same question again.\n")
	}
	return b.String()
}

func RunProvider(ctx context.Context, p Provider, task Task, prefs Preferences) (RunResult, error) {
	if strings.TrimSpace(task.ProjectPath) == "" {
		return RunResult{}, errors.New("project path is required for hands-on coding tasks")
	}
	remoteHarness := (p.ID == "openclaw" && (strings.TrimSpace(prefs.OpenClawSSHHost) != "" || strings.TrimSpace(prefs.OpenClawCommand) != "")) || p.ID == "workbench-runner"
	abs := task.ProjectPath
	var err error
	if !remoteHarness {
		abs, err = filepath.Abs(task.ProjectPath)
		if err != nil {
			return RunResult{}, err
		}
		if st, statErr := os.Stat(abs); statErr != nil || !st.IsDir() {
			return RunResult{}, fmt.Errorf("project path is not a directory: %s", abs)
		}
	}
	prompt := BuildWorkerPromptFromStoredKnowledge(task)

	var name string
	var args []string
	switch p.ID {
	case "antigravity":
		name = p.Command
		args = []string{"-p", prompt, "--mode", "accept-edits", "--sandbox", "--print-timeout", "30m"}
	case "gemini":
		name = p.Command
		args = []string{"-p", prompt, "--output-format", "json", "--approval-mode", "yolo", "--sandbox"}
	case "copilot":
		name = p.Command
		args = []string{"-p", prompt, "-s", "--no-ask-user"}
	case "claude":
		name = p.Command
		// acceptEdits handles repository writes non-interactively. A narrow
		// --allowedTools set covers common local verification commands without
		// granting unrestricted Bash; everything else remains provider-controlled
		// and can be routed around as a worker-local limitation.
		args = []string{"-p", prompt, "--output-format", "json", "--permission-mode", "acceptEdits", "--allowedTools"}
		args = append(args, claudeAllowedTools()...)
	case "codex":
		name = p.Command
		args = []string{"--ask-for-approval", "never", "exec", "--sandbox", "workspace-write", "--json", prompt}
	case "workbench-runner":
		res, runErr := RunClusterRunnerSSH(ctx, prefs.OpenClawSSHHost, task, prefs)
		if runErr == nil {
			res.Output = persistWorkerMemories(task, res.Output)
		}
		return res, runErr
	case "openclaw":
		if host := strings.TrimSpace(prefs.OpenClawSSHHost); host != "" {
			name = "ssh"
			args = []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=accept-new", host, "openclaw", "agent", "--message", prompt, "--headless"}
		} else if strings.TrimSpace(prefs.OpenClawCommand) != "" {
			res, runErr := runCommandTemplate(ctx, prefs.OpenClawCommand, abs, prompt)
			if runErr == nil {
				res.Output = persistWorkerMemories(task, res.Output)
			}
			return res, runErr
		} else if strings.TrimSpace(p.Command) != "" {
			name = p.Command
			args = []string{"agent", "--message", prompt, "--headless"}
		} else {
			return RunResult{}, errors.New("OpenClaw adapter is not configured")
		}
	case "ollama":
		res, runErr := runOllama(ctx, prompt)
		if runErr == nil {
			res.Output = persistWorkerMemories(task, res.Output)
		}
		return res, runErr
	default:
		return RunResult{}, fmt.Errorf("provider %s is not an executable worker", p.ID)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if !remoteHarness {
		cmd.Dir = abs
	}
	configureChildProcess(cmd, false)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if se := strings.TrimSpace(stderr.String()); se != "" {
		if out != "" {
			out += "\n\n"
		}
		out += se
	}
	out = normalizeProviderOutput(p.ID, out)
	res := classifyRunOutput(out)
	if strings.TrimSpace(res.WorkerUnavailable) != "" {
		res.Retryable = true
		return res, fmt.Errorf("%s is unavailable for this task: %s", p.Name, res.WorkerUnavailable)
	}
	if strings.TrimSpace(res.Attention) != "" && isWorkerSetupAttention(res.Attention) {
		q := res.Attention
		res.Attention = ""
		res.Retryable = true
		return res, fmt.Errorf("%s cannot act autonomously under its current local tool permissions: %s", p.Name, q)
	}
	if err != nil {
		low := strings.ToLower(out + " " + err.Error())
		res.Authentication = strings.Contains(low, "login") || strings.Contains(low, "sign in") || strings.Contains(low, "authenticate") || strings.Contains(low, "unauthorized") || strings.Contains(low, "credential")
		res.Retryable = res.Authentication || strings.Contains(low, "not found") || strings.Contains(low, "unknown option") || strings.Contains(low, "sandbox") || strings.Contains(low, "rate limit") || strings.Contains(low, "quota")
		if out == "" {
			out = err.Error()
			res.Output = out
		}
		return res, fmt.Errorf("%s exited with error: %w", p.Name, err)
	}
	res.Output = persistWorkerMemories(task, res.Output)
	return res, nil
}

func runCommandTemplate(ctx context.Context, template, project, prompt string) (RunResult, error) {
	qProject := shellQuote(project)
	qPrompt := shellQuote(prompt)
	line := strings.ReplaceAll(template, "{project}", qProject)
	line = strings.ReplaceAll(line, "{prompt}", qPrompt)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/D", "/S", "/C", line)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-lc", line)
	}
	cmd.Dir = project
	configureChildProcess(cmd, false)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	res := classifyRunOutput(buf.String())
	if err != nil {
		res.Retryable = true
		return res, err
	}
	return res, nil
}

func shellQuote(s string) string {
	if runtime.GOOS == "windows" {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func normalizeProviderOutput(providerID, out string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		return out
	}
	if providerID == "claude" || providerID == "gemini" {
		var v any
		if json.Unmarshal([]byte(out), &v) == nil {
			if s := extractText(v); strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	if providerID == "codex" {
		var texts []string
		sc := bufio.NewScanner(strings.NewReader(out))
		for sc.Scan() {
			var m map[string]any
			if json.Unmarshal([]byte(sc.Text()), &m) != nil {
				continue
			}
			if t := extractText(m); strings.TrimSpace(t) != "" {
				texts = append(texts, strings.TrimSpace(t))
			}
		}
		if len(texts) > 0 {
			return dedupeJoin(texts)
		}
	}
	return out
}

func extractText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var parts []string
		for _, y := range x {
			if s := extractText(y); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		preferred := []string{"result", "text", "content", "message", "output", "final", "assistant", "last_message"}
		for _, k := range preferred {
			if y, ok := x[k]; ok {
				if s := extractText(y); s != "" {
					return s
				}
			}
		}
		var keys []string
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s := extractText(x[k]); s != "" && len(s) > 30 {
				return s
			}
		}
	}
	return ""
}

func dedupeJoin(in []string) string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return strings.Join(out, "\n")
}

func classifyRunOutput(out string) RunResult {
	out = strings.TrimSpace(out)
	res := RunResult{Output: out}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "WORKER_UNAVAILABLE:") {
			res.WorkerUnavailable = strings.TrimSpace(line[len("WORKER_UNAVAILABLE:"):])
			if res.WorkerUnavailable == "" {
				res.WorkerUnavailable = "worker-local setup or tool policy prevented execution"
			}
			res.Retryable = true
			continue
		}
		if strings.HasPrefix(upper, "ATTENTION_REQUIRED:") {
			res.Attention = strings.TrimSpace(line[len("ATTENTION_REQUIRED:"):])
			if res.Attention == "" {
				res.Attention = "The worker needs a human decision."
			}
		}
	}
	return res
}

// isWorkerSetupAttention distinguishes a worker's local permission/configuration
// problem from a genuine product decision. A tool refusing shell/edit execution
// is not something the user should be paged for; Workbench should try another
// eligible worker. Explicit production/destructive/credential boundaries remain
// human attention even when the word "permission" appears.
func isWorkerSetupAttention(text string) bool {
	low := strings.ToLower(strings.TrimSpace(text))
	if low == "" {
		return false
	}
	humanBoundary := strings.Contains(low, "production") || strings.Contains(low, "deploy") || strings.Contains(low, "publish") || strings.Contains(low, "spend") || strings.Contains(low, "payment") || strings.Contains(low, "delete") || strings.Contains(low, "destroy") || strings.Contains(low, "credential") || strings.Contains(low, "secret")
	if humanBoundary {
		return false
	}
	toolSignal := strings.Contains(low, "tool") || strings.Contains(low, "bash") || strings.Contains(low, "shell") || strings.Contains(low, "command") || strings.Contains(low, "sandbox") || strings.Contains(low, "write") || strings.Contains(low, "edit") || strings.Contains(low, "toolchain")
	permissionSignal := strings.Contains(low, "permission mode") || strings.Contains(low, "tool calls are denied") || strings.Contains(low, "grant permission") || strings.Contains(low, "permission settings") || strings.Contains(low, "adjust settings") || strings.Contains(low, "not allowed to use") || strings.Contains(low, "requires approval") || strings.Contains(low, "interactive approval") || strings.Contains(low, "approval required") || strings.Contains(low, "requires interactive")
	return toolSignal && permissionSignal
}

func runOllama(ctx context.Context, prompt string) (RunResult, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:11434/api/tags", nil)
	resp, err := client.Do(req)
	if err != nil {
		return RunResult{Retryable: true}, err
	}
	defer resp.Body.Close()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&tags); err != nil || len(tags.Models) == 0 {
		return RunResult{Retryable: true}, errors.New("Ollama is online but no local model is available")
	}
	body, _ := json.Marshal(map[string]any{"model": tags.Models[0].Name, "prompt": prompt, "stream": false})
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:11434/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	client.Timeout = 30 * time.Minute
	resp, err = client.Do(req)
	if err != nil {
		return RunResult{Retryable: true}, err
	}
	defer resp.Body.Close()
	var gen struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&gen); err != nil {
		return RunResult{}, err
	}
	return classifyRunOutput(gen.Response), nil
}