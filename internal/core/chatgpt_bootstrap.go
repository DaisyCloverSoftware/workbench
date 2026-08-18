package core

import "strings"

const chatGPTBootstrapInstruction = `For software development, repository, server and cluster work, use Workbench as my execution bridge while keeping ChatGPT as the developer. Use my connected GitHub app to find my private repository whose name contains workbench-relay. If WORKBENCH_CAPABILITIES.json exists, read it first, then read WORKBENCH_CHATGPT.md from the repository root and follow that Workbench protocol and guide. ChatGPT writes and edits the code, handles Git and GitHub, creates/updates/merges pull requests, runs and diagnoses CI, and operates GitHub Actions. Use Workbench safe hands when bounded repository access is useful. Only pass to Workbench/OpenClaw machine-side work ChatGPT genuinely cannot execute itself, such as shell, systemd, Docker, Kubernetes, Helm, runner repair and deployment/runtime commands. OpenClaw is the operator, never the coder. Workbench must automatically re-engage OpenClaw if it stops with progress instead of verified completion. Never ask me to copy/paste prompts, watch OpenClaw, or type "continue". If an operation discovers that code, GitHub or CI changes are required, return that work to ChatGPT.`

// ChatGPTBootstrapInstruction is deliberately credential-free text that the
// desktop can put on the clipboard. It tells a fresh ChatGPT conversation how
// to discover the user's private Workbench relay through the already-connected
// GitHub app; it never includes MCP bearer values, hostnames or provider login
// material.
func ChatGPTBootstrapInstruction() string {
	return strings.TrimSpace(chatGPTBootstrapInstruction)
}
