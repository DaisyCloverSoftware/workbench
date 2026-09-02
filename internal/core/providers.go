package core

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func scanCommand(name string) (string, bool) {
	p, err := exec.LookPath(name)
	return p, err == nil
}

func commandOK(timeout time.Duration, name string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	configureChildProcess(cmd, false)
	return cmd.Run() == nil
}

func ollamaOnline() bool {
	client := http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get("http://127.0.0.1:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func ScanProviders() []Provider {
	providers := []Provider{}

	if p, ok := scanCommand("ollama"); ok {
		status := "installed · start Ollama to use"
		auth := false
		if ollamaOnline() {
			status = "online · zero marginal cost"
			auth = true
		}
		providers = append(providers, Provider{ID: "ollama", Name: "Local / Ollama", Capability: "local brain", Command: p, Installed: true, Authenticated: auth, Status: status, Cost: CostZero, Priority: 10, CanWrite: false, CanRunTools: false, Notes: "Cheap analysis, classification and first-pass reasoning."})
	} else {
		providers = append(providers, Provider{ID: "ollama", Name: "Local / Ollama", Capability: "local brain", Installed: false, Status: "not detected", Cost: CostZero, Priority: 10, Notes: "Optional local-first model runtime."})
	}

	if p, ok := scanCommand("ssh"); ok {
		providers = append(providers, Provider{ID: "workbench-runner", Name: "Workbench Cluster Runner", Capability: "remote coding control plane", Command: p, Installed: true, Authenticated: true, Status: "SSH transport detected · configure runner host", Cost: CostIncluded, Priority: 15, CanWrite: true, CanRunTools: true, Notes: "Preferred remote hands. Routes eligible work on the cluster under current task authority; it never infers OpenClaw authorization."})
	} else {
		providers = append(providers, Provider{ID: "workbench-runner", Name: "Workbench Cluster Runner", Capability: "remote coding control plane", Installed: false, Status: "SSH client not detected", Cost: CostIncluded, Priority: 15, CanWrite: true, CanRunTools: true, Notes: "Install an SSH client to use a remote Workbench Runner."})
	}

	if p, ok := scanCommand("agy"); ok {
		providers = append(providers, Provider{ID: "antigravity", Name: "Google Antigravity CLI", Capability: "coding agent / harness", Command: p, Installed: true, Authenticated: true, Status: "CLI detected · individual/subscription account capable", Cost: CostIncluded, Priority: 20, CanWrite: true, CanRunTools: true, Notes: "Current Google terminal agent. Routed before scarce Work/Codex; quota/eligibility is provider-managed."})
	} else {
		providers = append(providers, Provider{ID: "antigravity", Name: "Google Antigravity CLI", Capability: "coding agent / harness", Status: "not detected", Cost: CostIncluded, Priority: 20, CanWrite: true, CanRunTools: true, Notes: "Current Google terminal agent successor to Gemini CLI for individual accounts."})
	}

	if p, ok := scanCommand("gemini"); ok {
		providers = append(providers, Provider{ID: "gemini", Name: "Google Gemini CLI (legacy)", Capability: "enterprise/API coding agent", Command: p, Installed: true, Authenticated: true, Status: "legacy individual route retired · enterprise/API may work", Cost: CostMetered, Priority: 90, CanWrite: true, CanRunTools: true, Notes: "Individual-account terminal use moved to Antigravity CLI. Kept only for enterprise/API compatibility; metered routes stay opt-in."})
	} else {
		providers = append(providers, Provider{ID: "gemini", Name: "Google Gemini CLI (legacy)", Capability: "enterprise/API coding agent", Status: "not detected · use Antigravity for individual accounts", Cost: CostMetered, Priority: 90, CanWrite: true, CanRunTools: true})
	}

	if p, ok := scanCommand("copilot"); ok {
		providers = append(providers, Provider{ID: "copilot", Name: "GitHub Copilot CLI", Capability: "coding agent", Command: p, Installed: true, Authenticated: true, Status: "CLI detected · subscription capable", Cost: CostIncluded, Priority: 30, CanWrite: true, CanRunTools: true, Notes: "Uses the linked Copilot entitlement when authenticated."})
	} else {
		providers = append(providers, Provider{ID: "copilot", Name: "GitHub Copilot CLI", Capability: "coding agent", Status: "not detected", Cost: CostIncluded, Priority: 30, CanWrite: true, CanRunTools: true})
	}

	if p, ok := scanCommand("claude"); ok {
		authed := commandOK(2*time.Second, p, "auth", "status")
		status := "installed · sign-in required"
		if authed {
			status = "connected · subscription capable"
		}
		providers = append(providers, Provider{ID: "claude", Name: "Anthropic Claude Code", Capability: "coding agent / reviewer", Command: p, Installed: true, Authenticated: authed, Status: status, Cost: CostIncluded, Priority: 40, CanWrite: true, CanRunTools: true, Notes: "Good implementer/reviewer; subscription-backed use is preferred over metered API. Click Connect selected to open Claude's own sign-in flow."})
	} else {
		providers = append(providers, Provider{ID: "claude", Name: "Anthropic Claude Code", Capability: "coding agent / reviewer", Status: "not detected · install Claude Code CLI first", Cost: CostIncluded, Priority: 40, CanWrite: true, CanRunTools: true, Notes: "Official setup: install Node.js 18+ and Git for Windows, then run npm install -g @anthropic-ai/claude-code. After installation click Rescan, select Claude Code, then Connect selected. Claude App Pro/Max login is supported by Claude Code."})
	}

	if p, ok := findOpenClawExecutable(); ok {
		providers = append(providers, Provider{ID: "openclaw", Name: "OpenClaw", Capability: "owner-selected machine operations", Command: p, Installed: true, Authenticated: true, Status: "CLI detected · owner authorization required", Cost: CostIncluded, Priority: 50, CanWrite: true, CanRunTools: true, Notes: "OpenClaw is inert for automatic routing. Workbench may select it only for a task carrying durable explicit owner authorization naming OpenClaw."})
	} else {
		providers = append(providers, Provider{ID: "openclaw", Name: "OpenClaw", Capability: "owner-selected machine operations", Status: "not detected · optional explicit-use provider", Cost: CostIncluded, Priority: 50, CanWrite: true, CanRunTools: true, Notes: "Install OpenClaw only for deliberate owner-selected operations. Its absence does not affect direct Workbench machine controls."})
	}

	if p, ok := scanCommand("codex"); ok {
		authed := commandOK(2*time.Second, p, "login", "status")
		status := "installed · sign-in required"
		if authed {
			status = "connected · ChatGPT-managed auth"
		}
		providers = append(providers, Provider{ID: "codex", Name: "OpenAI Codex / Work", Capability: "coding agent", Command: p, Installed: true, Authenticated: authed, Status: status, Cost: CostScarce, Priority: 80, CanWrite: true, CanRunTools: true, Notes: "Powerful fallback. Routed late to conserve scarce Work/Codex usage."})
	} else {
		providers = append(providers, Provider{ID: "codex", Name: "OpenAI Codex / Work", Capability: "coding agent", Status: "not detected", Cost: CostScarce, Priority: 80, CanWrite: true, CanRunTools: true})
	}

	providers = append(providers,
		Provider{ID: "chatgpt", Name: "ChatGPT Chat", Capability: "lead brain via MCP", Installed: true, Authenticated: true, Status: "MCP bridge ready", Cost: CostIncluded, Priority: 5, CanWrite: false, CanRunTools: false, Notes: "Use Chat for brains. Workbench provides the hands through MCP without automatically consuming Work."},
		Provider{ID: "grok", Name: "xAI Grok", Capability: "chat / API", Installed: false, Status: "consumer chat adapter not automated", Cost: CostMetered, Priority: 100, Notes: "No brittle browser automation. Metered API remains opt-in."},
	)

	for i := range providers {
		providers[i].Status = strings.TrimSpace(providers[i].Status)
	}
	return providers
}

func LoginCommand(providerID string) (name string, args []string, ok bool) {
	switch providerID {
	case "codex":
		return "codex", []string{"login"}, true
	case "claude":
		return "claude", []string{"auth", "login"}, true
	case "copilot":
		return "copilot", []string{"login"}, true
	case "antigravity":
		return "agy", nil, true
	case "gemini":
		return "gemini", nil, true
	default:
		return "", nil, false
	}
}
