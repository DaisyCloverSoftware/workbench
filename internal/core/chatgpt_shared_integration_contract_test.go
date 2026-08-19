package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenAITunnelInstallerUsesProfileAndSecretReferences(t *testing.T) {
	text := installerScript(t, "install-openai-tunnel.sh")
	for _, want := range []string{
		"releases/latest/download",
		"current_client_usable",
		"init --help",
		"--sample sample_mcp_remote_no_auth",
		"--profile \"$profile\"",
		"--control-plane-api-key-ref \"file:$key_file\"",
		"MCP_EXTRA_HEADERS=\"Authorization: file:$mcp_auth_file\"",
		"doctor --profile \"$profile\" --explain",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("tunnel installer missing current secure profile contract %q", want)
		}
	}
	if strings.Contains(text, "releases/download/$version") {
		t.Fatal("tunnel installer must not depend on a guessed fallback release version")
	}
}

func TestChatGPTPluginPackagingScriptCreatesLocalBinding(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX packaging smoke")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Clean(filepath.Join(wd, "..", ".."))
	script := filepath.Join(repo, "scripts", "package-chatgpt-plugin.sh")
	home := t.TempDir()
	cmd := exec.Command("bash", script, "plugin_asdk_app_0123456789abcdef0123456789abcdef")
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("package script failed: %v\n%s", err, out)
	}
	plugin := filepath.Join(home, ".codex", "plugins", "workbench")
	app, err := os.ReadFile(filepath.Join(plugin, ".app.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(app), "plugin_asdk_app_0123456789abcdef0123456789abcdef") {
		t.Fatalf("generated app binding missing technical id: %s", app)
	}
	manifest, err := os.ReadFile(filepath.Join(plugin, ".codex-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"apps": "./.app.json"`) {
		t.Fatalf("generated manifest does not bind .app.json: %s", manifest)
	}
	market, err := os.ReadFile(filepath.Join(home, ".agents", "plugins", "marketplace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(market), `"path": "./.codex/plugins/workbench"`) {
		t.Fatalf("personal marketplace missing Workbench entry: %s", market)
	}
}

func TestChatGPTPluginPackagingScriptsRejectNonTechnicalIDs(t *testing.T) {
	text := installerScript(t, "package-chatgpt-plugin.sh")
	if !strings.Contains(text, "^plugin_asdk_app_") {
		t.Fatal("POSIX packager must validate the registered app technical id")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	psPath := filepath.Clean(filepath.Join(wd, "..", "..", "scripts", "package-chatgpt-plugin.ps1"))
	ps, err := os.ReadFile(psPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ps), "^plugin_asdk_app_") {
		t.Fatal("PowerShell packager must validate the registered app technical id")
	}
}
