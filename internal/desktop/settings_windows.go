//go:build windows

package desktop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/platform"
)

var loadedSettingsShell *Shell

func (s *Shell) refreshSettings(snapshot Snapshot) {
	prefs := s.eng.State().Preferences
	setChecked(s.controls[idProtectWork], prefs.AvoidWorkUsage)
	setChecked(s.controls[idAllowMetered], prefs.AllowMeteredAPI)
	s.refreshProviders()
	s.refreshSecrets()
	mcpStatus := "Local Chat/MCP bridge is unavailable."
	if s.mcpURL != "" {
		mcpStatus = s.mcpURL + "\r\nBearer authentication is enabled. The token is revealed only when you explicitly copy the connection details."
	} else if strings.TrimSpace(s.mcpErr) != "" {
		mcpStatus += " " + s.mcpErr
	}
	setWindowText(s.controls[idMCPStatus], mcpStatus)

	// Do not rewrite editable settings fields on every task/provider refresh.
	// Reload them only when Settings is first opened or the active project changes.
	// The shell identity sentinel matters when no project exists, because an empty
	// project ID is still a valid first-load state for global routing settings.
	if loadedSettingsShell == s && s.settingsProjectID == snapshot.ActiveProjectID {
		return
	}
	loadedSettingsShell = s
	s.settingsProjectID = snapshot.ActiveProjectID
	setWindowText(s.controls[idRunnerHost], prefs.OpenClawSSHHost)
	setWindowText(s.controls[idHarnessCommand], prefs.OpenClawCommand)
	setWindowText(s.controls[idNotifyCommand], prefs.NotificationCommand)
	setChecked(s.controls[idPublishReviews], false)
	setWindowText(s.controls[idReviewRemote], "")
	if snapshot.ActivePath == "" {
		return
	}
	policy, configured, err := core.PublicationPolicyFor(snapshot.ActivePath)
	if err != nil || !configured {
		return
	}
	if policy.Mode == core.PublicationPublish {
		setChecked(s.controls[idPublishReviews], true)
		setWindowText(s.controls[idReviewRemote], policy.RemoteURL)
	}
}

func (s *Shell) refreshProviders() {
	providers := s.eng.Providers()
	prefs := s.eng.State().Preferences
	procSendMessageW.Call(s.controls[idProviderList], lbResetContent, 0, 0)
	s.providerIDs = nil
	for _, provider := range providers {
		mark := "○"
		status := provider.Status
		if provider.Installed {
			mark = "●"
		}
		if provider.ID == "openclaw" && strings.TrimSpace(prefs.OpenClawSSHHost) != "" {
			mark = "●"
			status = "runner configured · " + prefs.OpenClawSSHHost
		}
		line := fmt.Sprintf("%s %s  ·  %s  ·  %s", mark, provider.Name, status, provider.Cost)
		ptr := wstr(line)
		procSendMessageW.Call(s.controls[idProviderList], lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
		s.providerIDs = append(s.providerIDs, provider.ID)
	}
}

func (s *Shell) refreshSecrets() {
	procSendMessageW.Call(s.controls[idSecretList], lbResetContent, 0, 0)
	for _, secret := range s.eng.State().Secrets {
		ptr := wstr("vault://" + secret.Name)
		procSendMessageW.Call(s.controls[idSecretList], lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
	}
}

func (s *Shell) handleSettingsCommand(id int, notify uint16) {
	switch id {
	case idProviderList:
		if notify == lbnSelChange {
			s.showSelectedProvider()
		}
	case idConnectProvider:
		s.connectSelectedProvider()
	case idRescanProviders:
		go s.eng.RescanProviders()
	case idCopyMCP:
		s.copyMCPConnection()
	case idSaveRouting:
		s.saveRoutingSettings()
	case idSaveReviewPolicy:
		s.saveReviewPolicy()
	case idSaveSecret:
		s.saveVaultSecret()
	case idRunUpdater:
		s.runUpdater()
	}
}

func (s *Shell) showSelectedProvider() {
	idx := listSelection(s.controls[idProviderList])
	if idx < 0 || idx >= len(s.providerIDs) {
		return
	}
	id := s.providerIDs[idx]
	for _, provider := range s.eng.Providers() {
		if provider.ID != id {
			continue
		}
		body := provider.Capability + "\r\n\r\n" + provider.Status
		if strings.TrimSpace(provider.Notes) != "" {
			body += "\r\n\r\n" + provider.Notes
		}
		messageBox(s.hwnd, provider.Name, body, mbOK|mbIconInformation)
		return
	}
}

func (s *Shell) connectSelectedProvider() {
	idx := listSelection(s.controls[idProviderList])
	if idx < 0 || idx >= len(s.providerIDs) {
		messageBox(s.hwnd, "AI workers", "Select a worker first.", mbOK|mbIconInformation)
		return
	}
	id := s.providerIDs[idx]
	if id == "openclaw" {
		host := strings.TrimSpace(s.eng.State().Preferences.OpenClawSSHHost)
		if host == "" {
			messageBox(s.hwnd, "Workbench Runner", "Enter the runner SSH host in Settings and save routing first.", mbOK|mbIconInformation)
			return
		}
		out, err := core.TestOpenClawSSH(host)
		if err != nil {
			messageBox(s.hwnd, "Runner connection failed", err.Error()+"\r\n\r\n"+out, mbOK|mbIconWarning)
			return
		}
		messageBox(s.hwnd, "Runner connected", "The configured remote harness responded over SSH.\r\n\r\n"+out, mbOK|mbIconInformation)
		return
	}
	if id == "chatgpt" {
		s.copyMCPConnection()
		return
	}
	if err := core.StartProviderLogin(id); err != nil {
		messageBox(s.hwnd, "Provider setup", providerSetupHint(id)+"\r\n\r\n"+err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Connect worker", "Workbench opened the provider's own sign-in flow. Finish sign-in, then click Rescan. Provider passwords are never entered into Workbench.", mbOK|mbIconInformation)
}

func providerSetupHint(id string) string {
	switch id {
	case "codex":
		return "Install the official Codex CLI, then connect it here."
	case "claude":
		return "Install Claude Code, then connect it here."
	case "copilot":
		return "Install GitHub Copilot CLI, then connect it here."
	case "antigravity":
		return "Install Google Antigravity CLI (agy), then connect it here."
	case "gemini":
		return "Gemini CLI is retained as a legacy enterprise/API adapter; individual Google accounts should prefer Antigravity CLI."
	case "grok":
		return "Workbench does not automate consumer browser sessions. Grok remains an adapter/API route rather than a browser-login worker."
	}
	return "No supported login adapter exists for this worker yet."
}

func (s *Shell) copyMCPConnection() {
	if s.mcpURL == "" {
		messageBox(s.hwnd, "Chat bridge unavailable", "The local MCP server could not start. "+s.mcpErr, mbOK|mbIconWarning)
		return
	}
	prefs := s.eng.State().Preferences
	text := "Workbench MCP\r\nURL: " + s.mcpURL + "\r\nAuthorization: Bearer " + prefs.MCPToken + "\r\n\r\nUse Chat for reasoning and Workbench for bounded hands. Delegate autonomous coding only when it is genuinely useful; poll durable task state without bothering the human for progress."
	if err := copyText(s.hwnd, text); err != nil {
		messageBox(s.hwnd, "Clipboard", err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Copied", "MCP connection details copied. Treat the bearer token as a local credential.", mbOK|mbIconInformation)
}

func (s *Shell) saveRoutingSettings() {
	prefs := s.eng.State().Preferences
	prefs.AvoidWorkUsage = isChecked(s.controls[idProtectWork])
	prefs.AllowMeteredAPI = isChecked(s.controls[idAllowMetered])
	prefs.OpenClawSSHHost = strings.TrimSpace(windowText(s.controls[idRunnerHost]))
	prefs.OpenClawCommand = strings.TrimSpace(windowText(s.controls[idHarnessCommand]))
	prefs.NotificationCommand = strings.TrimSpace(windowText(s.controls[idNotifyCommand]))
	if err := s.eng.SavePreferences(prefs); err != nil {
		messageBox(s.hwnd, "Cannot save routing", err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Routing saved", "Workbench will continue autonomously and protect scarce Work/Codex usage according to these settings.", mbOK|mbIconInformation)
}

func (s *Shell) saveReviewPolicy() {
	project, ok := s.eng.ActiveProject()
	if !ok {
		messageBox(s.hwnd, "Review delivery", "Select a project on the Work page first.", mbOK|mbIconInformation)
		return
	}
	mode := core.PublicationPrepare
	remote := ""
	if isChecked(s.controls[idPublishReviews]) {
		mode = core.PublicationPublish
		remote = strings.TrimSpace(windowText(s.controls[idReviewRemote]))
		if remote == "" {
			messageBox(s.hwnd, "Review delivery", "Enter the explicit review remote before enabling publication.", mbOK|mbIconInformation)
			return
		}
	}
	prefs := s.eng.State().Preferences
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := core.SavePublicationPolicyForExecutionHosts(ctx, project.Path, mode, remote, prefs.OpenClawSSHHost)
	if err != nil {
		if strings.TrimSpace(result.Local.Project) != "" {
			messageBox(s.hwnd, "Runner policy sync failed", err.Error()+"\r\n\r\nThe local policy was saved; the configured runner was not changed.", mbOK|mbIconWarning)
			return
		}
		messageBox(s.hwnd, "Review policy failed", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.settingsProjectID = ""
	loadedSettingsShell = nil
	scope := "Saved for local Workbench execution."
	if result.Runner != nil {
		scope = "Saved locally and synchronised to the configured Workbench runner."
	}
	message := scope + "\r\n\r\nCoding workers still cannot choose a remote, push branches or create pull requests directly."
	messageBox(s.hwnd, "Review policy saved", message, mbOK|mbIconInformation)
}

func (s *Shell) saveVaultSecret() {
	name := strings.TrimSpace(windowText(s.controls[idSecretName]))
	value := windowText(s.controls[idSecretValue])
	if name == "" || value == "" {
		messageBox(s.hwnd, "Vault", "Enter both a secret name and value.", mbOK|mbIconInformation)
		return
	}
	ciphertext, err := platform.ProtectString(value)
	if err != nil {
		messageBox(s.hwnd, "Vault encryption failed", err.Error(), mbOK|mbIconWarning)
		return
	}
	if err := s.eng.AddSecret(core.SecretRef{Name: name, Ciphertext: ciphertext, CreatedAt: time.Now()}); err != nil {
		messageBox(s.hwnd, "Vault save failed", err.Error(), mbOK|mbIconWarning)
		return
	}
	setWindowText(s.controls[idSecretName], "")
	setWindowText(s.controls[idSecretValue], "")
	messageBox(s.hwnd, "Secret stored", "Encrypted as vault://"+name+" with Windows DPAPI. The raw value is not exposed to AI tools.", mbOK|mbIconInformation)
}

func (s *Shell) runUpdater() {
	exe, err := os.Executable()
	if err != nil {
		messageBox(s.hwnd, "Updater", err.Error(), mbOK|mbIconWarning)
		return
	}
	updater := filepath.Join(filepath.Dir(exe), "Workbench-Updater.exe")
	if info, statErr := os.Stat(updater); statErr != nil || info.IsDir() {
		messageBox(s.hwnd, "Updater not installed", "Workbench-Updater.exe was not found next to Workbench.exe. The verified updater is included in official Workbench release packages.", mbOK|mbIconInformation)
		return
	}
	cmd := exec.Command(updater)
	if err := cmd.Start(); err != nil {
		messageBox(s.hwnd, "Cannot start updater", err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Updater opened", "The separate verified updater will check the official Workbench release and only replace Workbench.exe after checksum and executable validation.", mbOK|mbIconInformation)
}
