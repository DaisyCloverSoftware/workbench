//go:build windows

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFindUnrealEditorCmdUsesLauncherManifestAndNewestVersion(t *testing.T) {
	olderExe := unrealTestInstall(t, 5, 9, 3)
	newerExe := unrealTestInstall(t, 5, 10, 0)
	installRoot := func(executable string) string {
		return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(executable))))
	}
	programData := t.TempDir()
	manifestDir := filepath.Join(programData, "Epic", "UnrealEngineLauncher")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := epicLauncherInstallManifest{}
	manifest.InstallationList = append(manifest.InstallationList,
		struct {
			InstallLocation string `json:"InstallLocation"`
			AppName         string `json:"AppName"`
		}{InstallLocation: installRoot(olderExe), AppName: "UE_5.9"},
		struct {
			InstallLocation string `json:"InstallLocation"`
			AppName         string `json:"AppName"`
		}{InstallLocation: installRoot(newerExe), AppName: "UE_5.10"},
	)
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "LauncherInstalled.dat"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ProgramData", programData)
	t.Setenv("ProgramW6432", "")
	t.Setenv("ProgramFiles", "")
	t.Setenv("PATH", "")

	if got := findUnrealEditorCmdExecutable(); got != newerExe {
		t.Fatalf("selected Unreal executable %q, want %q", got, newerExe)
	}
}
