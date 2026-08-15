package core

import "strings"

const StructuredHarnessProviderID = "structured-harness"

func configuredHarnessProvider(prefs Preferences) (Provider, bool) {
	path := strings.TrimSpace(prefs.HarnessAdapterPath)
	if path == "" {
		return Provider{}, false
	}
	resolved, err := validateHarnessAdapterPath(path)
	if err != nil {
		return Provider{}, false
	}
	return Provider{
		ID:            StructuredHarnessProviderID,
		Name:          "Structured Harness Adapter",
		Capability:    "external coding harness",
		Command:       resolved,
		Installed:     true,
		Authenticated: true,
		Status:        "configured · structured protocol v1",
		Cost:          CostIncluded,
		Priority:      45,
		CanWrite:      true,
		CanRunTools:   true,
		Notes:         "Operator-configured executable using bounded JSON jobs. Workbench retains review/publication authority.",
	}, true
}

func providerInventoryWithConfiguredHarness(base []Provider, prefs Preferences) []Provider {
	out := append([]Provider(nil), base...)
	if provider, ok := configuredHarnessProvider(prefs); ok {
		out = append(out, provider)
	} else if strings.TrimSpace(prefs.HarnessAdapterPath) != "" {
		out = append(out, Provider{
			ID:         StructuredHarnessProviderID,
			Name:       "Structured Harness Adapter",
			Capability: "external coding harness",
			Installed:  false,
			Status:     "configured adapter executable is unavailable",
			Cost:       CostIncluded,
			Priority:   45,
			CanWrite:   true,
			CanRunTools: true,
			Notes:      "Choose one existing adapter executable; Workbench does not run this setting through a shell.",
		})
	}
	if strings.TrimSpace(prefs.OpenClawCommand) != "" {
		out = append(out, Provider{
			ID:         "legacy-harness-command",
			Name:       "Legacy harness command",
			Capability: "migration required",
			Installed:  false,
			Status:     "disabled · migrate to a structured adapter executable",
			Cost:       CostIncluded,
			Priority:   46,
			CanWrite:   false,
			CanRunTools: false,
			Notes:      "Legacy shell-template harness commands are preserved in state for migration but are never eligible to execute.",
		})
	}
	return out
}
