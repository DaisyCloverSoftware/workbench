//go:build windows

package desktop

import (
	"errors"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestRunnerConnectionProviderInfoExposesOnlySafeAuthenticationCategory(t *testing.T) {
	provider := runnerConnectionProviderInfo(core.ErrRunnerSSHAuthentication)
	if provider.ID != core.RunnerConnectionProviderID || provider.Ready || !provider.Installed {
		t.Fatalf("unexpected connection provider: %#v", provider)
	}
	if !strings.Contains(strings.ToLower(provider.Status), "unattended ssh authentication") {
		t.Fatalf("status=%q does not explain unattended SSH requirement", provider.Status)
	}
}

func TestRunnerConnectionProviderInfoDoesNotEchoUnknownRawFailure(t *testing.T) {
	provider := runnerConnectionProviderInfo(errors.New("secret-key-material should never be displayed"))
	if strings.Contains(provider.Status, "secret-key-material") {
		t.Fatalf("raw transport detail leaked into provider status: %q", provider.Status)
	}
}
