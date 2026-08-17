package core

import "strings"

const chatGPTBootstrapInstruction = `For software-development, repository, server and cluster work, use Workbench as my default execution layer. Use my connected GitHub app to find my private repository whose name contains workbench-relay. If WORKBENCH_CAPABILITIES.json exists, read it first, then read WORKBENCH_CHATGPT.md from the repository root and follow that Workbench protocol and guide. ChatGPT is the primary brain/coder: do the reasoning and exact coding in this conversation, use Workbench safe hands before autonomous delegation, and do not ask me to copy/paste prompts into OpenClaw, Codex, terminals or other chats when Workbench can carry the operation. Preserve Workbench routing, keep scarce Codex/Work as a last resort, and leave metered API fallback opt-in. If Workbench is unavailable, tell me what is unavailable rather than silently turning me into the message shuttle.`

// ChatGPTBootstrapInstruction is deliberately credential-free text that the
// desktop can put on the clipboard. It tells a fresh ChatGPT conversation how
// to discover the user's private Workbench relay through the already-connected
// GitHub app; it never includes MCP bearer values, hostnames or provider login
// material.
func ChatGPTBootstrapInstruction() string {
	return strings.TrimSpace(chatGPTBootstrapInstruction)
}
