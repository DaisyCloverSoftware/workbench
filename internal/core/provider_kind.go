package core

// IsCodingWorkerProvider reports whether a provider is a hands-on coding worker
// rather than an integration, local reasoning-only service, or another
// Workbench control plane. It intentionally says nothing about current
// installation/authentication state so Settings can show unavailable workers
// on the host where they would execute.
func IsCodingWorkerProvider(p Provider) bool {
	if !p.CanWrite || !p.CanRunTools {
		return false
	}
	switch p.ID {
	case "workbench-runner", "legacy-harness-command":
		return false
	default:
		return true
	}
}

func ProviderReadyForCoding(p Provider) bool {
	return IsCodingWorkerProvider(p) && p.Installed && p.Authenticated && p.Command != ""
}
