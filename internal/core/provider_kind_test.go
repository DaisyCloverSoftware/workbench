package core

import "testing"

func TestCodingWorkerClassificationSeparatesControlPlanesAndIntegrations(t *testing.T) {
	cases := []struct {
		provider Provider
		want     bool
	}{
		{Provider{ID: "claude", CanWrite: true, CanRunTools: true}, true},
		{Provider{ID: StructuredHarnessProviderID, CanWrite: true, CanRunTools: true}, true},
		{Provider{ID: "workbench-runner", CanWrite: true, CanRunTools: true}, false},
		{Provider{ID: "chatgpt", CanWrite: false, CanRunTools: false}, false},
		{Provider{ID: "ollama", CanWrite: false, CanRunTools: false}, false},
	}
	for _, tc := range cases {
		if got := IsCodingWorkerProvider(tc.provider); got != tc.want {
			t.Fatalf("IsCodingWorkerProvider(%q)=%v want %v", tc.provider.ID, got, tc.want)
		}
	}
}

func TestProviderReadyForCodingRequiresInstallAuthAndCommand(t *testing.T) {
	provider := Provider{ID: "claude", CanWrite: true, CanRunTools: true, Installed: true, Authenticated: true, Command: "claude"}
	if !ProviderReadyForCoding(provider) {
		t.Fatal("fully configured coding worker should be ready")
	}
	provider.Authenticated = false
	if ProviderReadyForCoding(provider) {
		t.Fatal("unauthenticated coding worker reported ready")
	}
}

func TestRunnerProviderLoginRejectsUnsupportedProviderBeforeSSH(t *testing.T) {
	if err := StartRunnerProviderLogin("runner.example", "grok"); err == nil {
		t.Fatal("unsupported provider login should be rejected")
	}
}
