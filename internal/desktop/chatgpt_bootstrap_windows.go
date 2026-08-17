//go:build windows

package desktop

import "github.com/DaisyCloverSoftware/workbench/internal/core"

const idCopyChatGPTBootstrap = 3228

func (s *Shell) createChatGPTBootstrapControl() {
	// This button copies only the credential-free discovery instruction. The MCP
	// connection button remains separate because its clipboard payload includes a
	// local bearer credential and must never be confused with fresh-chat setup.
	s.control(idCopyChatGPTBootstrap, "BUTTON", "Copy ChatGPT bootstrap", wsChild|wsTabStop|bsPushButton)
}

func (s *Shell) handleChatGPTBootstrapCommand(id int) bool {
	if id != idCopyChatGPTBootstrap {
		return false
	}
	if err := copyText(s.hwnd, core.ChatGPTBootstrapInstruction()); err != nil {
		messageBox(s.hwnd, "Clipboard", err.Error(), mbOK|mbIconWarning)
		return true
	}
	messageBox(s.hwnd, "Copied", "Credential-free ChatGPT bootstrap copied. Paste it once into ChatGPT instructions when you want fresh chats to discover and use your private Workbench relay automatically.", mbOK|mbIconInformation)
	return true
}
