//go:build windows

package desktop

import "github.com/DaisyCloverSoftware/workbench/internal/core"

// Keep the bridge on the last successfully persisted routing target, even if
// the engine tentatively changes its in-memory preferences before Save fails.
func (s *Shell) saveRoutingPreferences(prefs core.Preferences) error {
	persist := func() error { return s.eng.SavePreferences(prefs) }
	if s.hostBridgeTarget == nil {
		// Legacy shells without an owned bridge retain normal preference saving.
		return persist()
	}
	return s.hostBridgeTarget.Save(prefs.OpenClawSSHHost, persist)
}
