package main

import _ "embed"

const privateChatGuidePath = "WORKBENCH_CHATGPT.md"

// privateChatGuide is published only by the private relay. It is deliberately
// non-secret and self-describes the safe-hands/autonomous transport so a fresh
// ChatGPT conversation can discover Workbench without making the user shuttle
// prompts between chats, providers or terminals.
//
//go:embed WORKBENCH_CHATGPT.md
var privateChatGuide []byte
