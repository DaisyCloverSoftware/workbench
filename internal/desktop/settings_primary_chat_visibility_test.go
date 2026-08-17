package desktop

import (
	"os"
	"strings"
	"testing"
)

func TestSettingsKeepsPrimaryChatVisibleOutsideScrollableInventory(t *testing.T) {
	controls, err := os.ReadFile("production_controls_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	controlText := string(controls)
	for _, want := range []string{
		`idSettingsTitle:    "PRIMARY · This PC · ChatGPT Chat · normal Workbench brain"`,
		`idProvidersLabel:   "Chat-first routing · primary brain + autonomous workers"`,
	} {
		if !strings.Contains(controlText, want) {
			t.Fatalf("production Settings controls do not pin primary ChatGPT: missing %q", want)
		}
	}
	if strings.Contains(controlText, `setWindowText(s.controls[idSettingsTitle], "")`) {
		t.Fatal("production Settings must not hide the pinned primary ChatGPT status")
	}

	layout, err := os.ReadFile("production_layout_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	layoutText := string(layout)
	for _, want := range []string{
		`moveWindow(s.controls[idSettingsTitle], x+4, top+2, left-8, 20)`,
		`moveWindow(s.controls[idProvidersLabel], x+4, top+28, left-8, 20)`,
		`moveWindow(s.controls[idProviderList], x+4, top+52, left-8, 106)`,
	} {
		if !strings.Contains(layoutText, want) {
			t.Fatalf("production Settings layout can hide primary ChatGPT: missing %q", want)
		}
	}
}
