package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestProductionSettingsOwnsCredentialFreeChatGPTBootstrapAction(t *testing.T) {
	checks := map[string][]string{
		"chatgpt_bootstrap_windows.go": {
			`const idCopyChatGPTBootstrap = 3228`,
			`"Copy ChatGPT bootstrap"`,
			`core.ChatGPTBootstrapInstruction()`,
		},
		"production_shell_windows.go": {
			`s.createChatGPTBootstrapControl()`,
			`s.handleChatGPTBootstrapCommand(id)`,
			`showWindow(s.controls[idCopyChatGPTBootstrap], s.page == pageSettings)`,
		},
		"production_layout_windows.go": {
			`moveWindow(s.controls[idCopyChatGPTBootstrap]`,
		},
		"production_buttons_windows.go": {
			`idCopyMCP, idCopyChatGPTBootstrap, idSaveRouting`,
		},
	}
	for path, wants := range checks {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(b)
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing ChatGPT bootstrap integration %q", path, want)
			}
		}
	}
}
