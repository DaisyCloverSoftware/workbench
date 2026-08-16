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

func (s *Shell) invalidateSettingsCache() {
	if loadedSettingsShell == s {
		loadedSettingsShell = nil
	}
	s.settingsProjectID = ""
}

func (s *Shell) refreshSettings(snapshot Snapshot) {
	// Settings contains native listboxes plus filesystem-backed provider and
	// policy state. Rebuilding all of it for every engine notification or every
	// page revisit is unnecessary and, on real Windows desktops, can accumulate
	// enough synchronous UI work to starve the message pump. The active project,
	// explicit settings mutations and asynchronous runner-inventory generation
	// invalidate this cache only when the displayed state can genuinely change.
	runnerGeneration := runnerProviderGeneration.Load()
	if loadedSettingsShell == s && s.settingsProjectID == snapshot.ActiveProjectID && loadedSettingsRunnerProviderGeneration == runnerGeneration {
		return
	}
	loadedSettingsShell = s
	s.settingsProjectID = snapshot.ActiveProjectID
	loadedSettingsRunnerProviderGeneration = runnerGeneration

	prefs := s.eng.State().Preferences
	setChecked(s.controls[idProtectWork], prefs.AvoidWorkUsage)
	setChecked(s.controls[idAllowMetered], prefs.AllowMeteredAPI)
	setWindowText(s.controls[idHarnessLabel], "Structured harness adapter executable (optional)")
	cueBanner(s.controls[idHarnessCommand], "absolute path to one adapter executable; no arguments or shell placeholders")
	s.refreshProviders()
	s.refreshSecrets()

	mcpStatus := "Local Chat/MCP bridge is unavailable."
	if s.mcpURL != "" {
		mcpStatus = s.mcpURL + "\r\nBearer authentication is enabled. This means the local bridge is ready; it does not claim that a specific ChatGPT conversation is attached."
	} else if strings.TrimSpace(s.mcpErr) != "" {
		mcpStatus += " " + s.mcpErr
	}
	setWindowText(s.controls[idMCPStatus], mcpStatus)

	setWindowText(s.controls[idRunnerHost], prefs.OpenClawSSHHost)
	setWindowText(s.controls[idHarnessCommand], prefs.HarnessAdapterPath)
	setWindowText(s.controls[idNotifyCommand], prefs.NotificationCommand)
	setChecked(s.controls[idPublishReviews], false)
	setWindowText(s.controls[idReviewRemote], "")
	if snapshot.ActivePath == "" {
		return
	}
	// Active projects were already canonicalised when registered. Reading the
	// local policy/mirror must therefore not invoke Git or SSH on the UI thread.
	policy, configured, err := core.PublicationPolicyForKnownProject(snapshot.ActivePath)
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
	host := strings.TrimSpace(prefs.OpenClawSSHHost)
	if host != "" {
		s.ensureRunnerProviderInventory(false)
	}
	runner := runnerProviderInventory(host)
	procSendMessageW.Call(s.controls[idProviderList], lbResetContent, 0, 0)
	s.providerIDs = nil

	appendLine := func(target, line string) {
		ptr := wstr(line)
		procSendMessageW.Call(s.controls[idProviderList], lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
		s.providerIDs = append(s.providerIDs, target)
	}
	appendLocal := func() {
		for _, provider := range providers {
			if !core.IsCodingWorkerProvider(provider) {
				continue
			}
			mark := "○"
			if core.ProviderReadyForCoding(provider) {
				mark = "●"
			}
			line := fmt.Sprintf("%s This PC · %s  ·  %s  ·  %s", mark, provider.Name, provider.Status, provider.Cost)
			appendLine("local:"+provider.ID, line)
		}
	}
	appendRunner := func() {
		if host == "" {
			return
		}
		if len(runner.Providers) == 0 {
			status := "no coding workers detected"
			if runner.Loading {
				status = "scanning coding workers…"
			} else if runner.Failed {
				status = "worker inventory unavailable"
			}
			appendLine("status:runner", "○ Runner · "+status)
			return
		}
		for _, provider := range runner.Providers {
			mark := "○"
			if provider.Ready {
				mark = "●"
			}
			line := fmt.Sprintf("%s Runner · %s  ·  %s  ·  %s", mark, provider.Name, provider.Status, provider.Cost)
			appendLine("runner:"+provider.ID, line)
		}
	}

	preferRunner := false
	if project, ok := s.eng.ActiveProject(); ok {
		preferRunner = core.IsRunnerProjectReference(project.Path)
	}
	if preferRunner {
		appendRunner()
		appendLocal()
	} else {
		appendLocal()
		appendRunner()
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
		s.invalidateSettingsCache()
		resetRunnerProviderInventory()
		s.ensureRunnerProviderInventory(true)
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
	scope, id := providerListTarget(s.providerIDs[idx])
	if scope == "status" {
		messageBox(s.hwnd, "Cluster runner", "Workbench has not received a usable coding-worker inventory from the configured runner yet. Rescan after the runner is reachable and current.", mbOK|mbIconInformation)
		return
	}
	if scope == "runner" {
		host := strings.TrimSpace(s.eng.State().Preferences.OpenClawSSHHost)
		provider, ok := runnerProviderByID(host, id)
		if !ok {
			return
		}
		body := "Execution host: Cluster runner\r\n\r\n" + provider.Capability + "\r\n\r\n" + provider.Status
		messageBox(s.hwnd, provider.Name, body, mbOK|mbIconInformation)
		return
	}
	for _, provider := range s.eng.Providers() {
		if provider.ID != id {
			continue
		}
		body := "Execution host: This PC\r\n\r\n" + provider.Capability + "\r\n\r\n" + provider.Status
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
		messageBox(s.hwnd, "Coding workers", "Select a worker first.", mbOK|mbIconInformation)
		return
	}
	scope, id := providerListTarget(s.providerIDs[idx])
	if scope == "status" {
		messageBox(s.hwnd, "Cluster runner", "There is no selectable runner worker in this row. Click Rescan after the runner is reachable.", mbOK|mbIconInformation)
		return
	}
	if scope == "runner" {
		host := strings.TrimSpace(s.eng.State().Preferences.OpenClawSSHHost)
		provider, ok := runnerProviderByID(host, id)
		if !ok {
			messageBox(s.hwnd, "Runner worker", "Workbench no longer has this runner worker in its current inventory. Click Rescan.", mbOK|mbIconInformation)
			return
		}
		if provider.Ready {
			messageBox(s.hwnd, "Runner worker ready", provider.Name+" is ready on the configured cluster runner.", mbOK|mbIconInformation)
			return
		}
		if !provider.Installed {
			messageBox(s.hwnd, "Runner worker not installed", runnerProviderSetupHint(id), mbOK|mbIconInformation)
			return
		}
		if err := core.StartRunnerProviderLogin(host, id); err != nil {
			messageBox(s.hwnd, "Runner worker setup", "Workbench could not open the provider's allowlisted login flow on the runner.\r\n\r\n"+err.Error(), mbOK|mbIconWarning)
			return
		}
		messageBox(s.hwnd, "Connect runner worker", "Workbench opened a human-visible SSH console for the provider's own login flow on the cluster runner. Finish provider sign-in, close the console when it completes, then click Rescan. Passwords and OAuth tokens are not entered into Workbench.", mbOK|mbIconInformation)
		return
	}

	switch id {
	case core.StructuredHarnessProviderID:
		for _, provider := range s.eng.Providers() {
			if provider.ID != id {
				continue
			}
			if !provider.Installed {
				messageBox(s.hwnd, "Structured harness adapter", "Choose one existing adapter executable in Settings and save routing. Workbench passes a bounded JSON job on stdin and never runs this setting through a command shell.", mbOK|mbIconInformation)
				return
			}
			messageBox(s.hwnd, "Structured harness ready", "The configured adapter executable is available on This PC. Workbench will use structured protocol v1 inside an isolated task workspace and will retain review/publication authority.", mbOK|mbIconInformation)
			return
		}
		return
	case "openclaw":
		messageBox(s.hwnd, "OpenClaw", "This row describes the OpenClaw CLI on This PC. Configure/authenticate it through its own CLI if required, then click Rescan. Cluster-project execution uses the separate runner inventory shown in Runner rows.", mbOK|mbIconInformation)
		return
	}
	if err := core.StartProviderLogin(id); err != nil {
		messageBox(s.hwnd, "Provider setup", providerSetupHint(id)+"\r\n\r\n"+err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Connect worker", "Workbench opened the provider's own sign-in flow on This PC. Finish sign-in, then click Rescan. Provider passwords are never entered into Workbench.", mbOK|mbIconInformation)
}

func providerSetupHint(id string) string {
	switch id {
	case "codex":
		return "Install the official Codex CLI on This PC, then connect it here."
	case "claude":
		return "Install Node.js 18+ and Git for Windows, then run: npm install -g @anthropic-ai/claude-code. After installation click Rescan, then Connect selected."
	case "copilot":
		return "Install GitHub Copilot CLI on This PC, then connect it here."
	case "antigravity":
		return "Install Google Antigravity CLI (agy) on This PC, then connect it here."
	case "gemini":
		return "Gemini CLI is retained as a legacy enterprise/API adapter; individual Google accounts should prefer Antigravity CLI."
	case "openclaw":
		return "Install and configure the OpenClaw CLI on This PC. Workbench invokes the detected CLI with fixed arguments; it does not accept an OpenClaw shell command template."
	}
	return "No supported local login adapter exists for this worker yet."
}

func runnerProviderSetupHint(id string) string {
	switch id {
	case "claude":
		return "Anthropic Claude Code is not installed on the cluster runner. This PC's Claude installation is a separate worker and cannot edit a runner:// project. Workbench will continue to use other detected runner workers until Claude Code is installed on that execution host."
	case "codex":
		return "OpenAI Codex is not installed on the cluster runner. This PC's Codex installation is separate from runner:// project execution."
	case "copilot":
		return "GitHub Copilot CLI is not installed on the cluster runner."
	case "antigravity":
		return "Google Antigravity CLI is not installed on the cluster runner."
	default:
		return "This coding worker is not installed on the cluster runner. Workbench will not pretend a This PC installation can execute a runner:// project."
	}
}

func (s *Shell) copyMCPConnection() {
	if s.mcpURL == "" {
		messageBox(s.hwnd, "Chat bridge unavailable", "The local MCP server could not start. "+s.mcpErr, mbOK|mbIconWarning)
		return
	}
	prefs := s.eng.State().Preferences
	text := "Workbench MCP\r\nURL: " + s.mcpURL + "\r\nAuthorization: Bearer " + prefs.MCPToken + "\r\n\r\nThis proves the local Workbench bridge is ready. Whether a particular ChatGPT client can attach to this endpoint depends on that client's MCP/app connection model."
	if err := copyText(s.hwnd, text); err != nil {
		messageBox(s.hwnd, "Clipboard", err.Error(), mbOK|mbIconWarning)
		return
	}
	messageBox(s.hwnd, "Copied", "MCP bridge connection details copied. Treat the bearer token as a local credential.", mbOK|mbIconInformation)
}

func (s *Shell) saveRoutingSettings() {
	prefs := s.eng.State().Preferences
	oldRunnerHost := strings.TrimSpace(prefs.OpenClawSSHHost)
	prefs.AvoidWorkUsage = isChecked(s.controls[idProtectWork])
	prefs.AllowMeteredAPI = isChecked(s.controls[idAllowMetered])
	prefs.OpenClawSSHHost = strings.TrimSpace(windowText(s.controls[idRunnerHost]))
	adapter := strings.TrimSpace(windowText(s.controls[idHarnessCommand]))
	if adapter != "" {
		resolved, err := core.ValidateHarnessAdapterPath(adapter)
		if err != nil {
			messageBox(s.hwnd, "Structured harness adapter", "Workbench did not save the adapter path because it is not one existing executable file.\r\n\r\n"+err.Error(), mbOK|mbIconWarning)
			return
		}
		adapter = resolved
	}
	legacyCleared := strings.TrimSpace(prefs.OpenClawCommand) != ""
	prefs.HarnessAdapterPath = adapter
	prefs.OpenClawCommand = ""
	prefs.NotificationCommand = strings.TrimSpace(windowText(s.controls[idNotifyCommand]))
	s.invalidateSettingsCache()
	if err := s.eng.SavePreferences(prefs); err != nil {
		messageBox(s.hwnd, "Cannot save routing", err.Error(), mbOK|mbIconWarning)
		return
	}
	if oldRunnerHost != strings.TrimSpace(prefs.OpenClawSSHHost) {
		resetRunnerProviderInventory()
	}
	setWindowText(s.controls[idHarnessCommand], adapter)
	message := "Workbench will continue autonomously and protect scarce Work/Codex usage according to these settings."
	if legacyCleared {
		message += "\r\n\r\nThe obsolete shell-template harness command was disabled and removed from saved routing settings."
	}
	messageBox(s.hwnd, "Routing saved", message, mbOK|mbIconInformation)
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

	if core.IsRunnerProjectReference(project.Path) {
		host := strings.TrimSpace(prefs.OpenClawSSHHost)
		if host == "" {
			messageBox(s.hwnd, "Review policy failed", "This cluster project needs a configured Workbench Runner SSH host before its review policy can be saved.", mbOK|mbIconWarning)
			return
		}
		action := "prepare"
		if mode == core.PublicationPublish {
			action = "publish"
		}
		_, err := core.RunRunnerPublicationPolicySSH(ctx, host, core.RunnerPolicyRequest{Action: action, Project: project.Path, RemoteURL: remote})
		if err != nil {
			messageBox(s.hwnd, "Runner review policy failed", err.Error(), mbOK|mbIconWarning)
			return
		}
		if _, err := core.SaveRunnerProjectPublicationPolicyMirror(core.PublicationPolicy{Project: project.Path, Mode: mode, RemoteURL: remote}); err != nil {
			messageBox(s.hwnd, "Review policy saved on runner", "The cluster runner accepted the policy, but Workbench could not save its local display mirror.\r\n\r\n"+err.Error(), mbOK|mbIconWarning)
			return
		}
		s.invalidateSettingsCache()
		messageBox(s.hwnd, "Review policy saved", "Saved on the configured Workbench runner. The desktop keeps only an operator-side mirror so Settings and verified GitHub review links remain usable; coding workers still cannot choose a remote or publish directly.", mbOK|mbIconInformation)
		return
	}

	result, err := core.SavePublicationPolicyForExecutionHosts(ctx, project.Path, mode, remote, prefs.OpenClawSSHHost)
	if err != nil {
		if strings.TrimSpace(result.Local.Project) != "" {
			messageBox(s.hwnd, "Runner policy sync failed", err.Error()+"\r\n\r\nThe local policy was saved; the configured runner was not changed.", mbOK|mbIconWarning)
			return
		}
		messageBox(s.hwnd, "Review policy failed", err.Error(), mbOK|mbIconWarning)
		return
	}
	s.invalidateSettingsCache()
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
	s.invalidateSettingsCache()
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
