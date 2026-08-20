package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func unrealTestInstall(t *testing.T, major, minor, patch int) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "UE_Test")
	executable := filepath.Join(root, "Engine", "Binaries", "Win64", "UnrealEditor-Cmd.exe")
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("test-only"), 0o600); err != nil {
		t.Fatal(err)
	}
	buildDir := filepath.Join(root, "Engine", "Build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte(fmt.Sprintf("{\"MajorVersion\":%d,\"MinorVersion\":%d,\"PatchVersion\":%d,\"Changelist\":12345}", major, minor, patch))
	if err := os.WriteFile(filepath.Join(buildDir, "Build.version"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return executable
}

func TestUnrealBuildVersionIsReadFromValidatedEngineLayout(t *testing.T) {
	executable := unrealTestInstall(t, 5, 6, 1)
	version, err := readUnrealBuildVersion(executable)
	if err != nil {
		t.Fatal(err)
	}
	if got := formatUnrealVersion(version); got != "Unreal Engine 5.6.1" {
		t.Fatalf("version=%q", got)
	}
	if _, err := unrealVersionFileForExecutable(filepath.Join(filepath.Dir(executable), "cmd.exe")); err == nil {
		t.Fatal("non-Unreal executable was accepted")
	}
	if _, err := unrealVersionFileForExecutable(filepath.Join(t.TempDir(), "UnrealEditor-Cmd.exe")); err == nil {
		t.Fatal("Unreal executable outside Engine/Binaries/Win64 was accepted")
	}
}

func TestUnrealSmokeInvocationUsesFixedProjectQuitAndDisablesActiveScripting(t *testing.T) {
	executable := unrealTestInstall(t, 5, 7, 0)
	project := filepath.Join(t.TempDir(), "WorkbenchSmoke.uproject")
	if err := os.WriteFile(project, []byte(unrealSmokeProjectDocument), 0o600); err != nil {
		t.Fatal(err)
	}
	name, args, err := unrealSmokeInvocation(executable, project)
	if err != nil {
		t.Fatal(err)
	}
	if name != executable {
		t.Fatalf("name=%q", name)
	}
	want := []string{
		project,
		"-ExecCmds=Quit",
		"-unattended",
		"-stdout",
		"-nop4",
		"-nullrhi",
		"-nosplash",
		"-nowrite",
		"-NOAUTOINIUPDATE",
		"-norecentproject",
		"-NoAssetRegistryCache",
		"-NoAssetRegistryCacheWrite",
		"-NoShaderCompile",
		"-NoZenAutoLaunch",
		"-DisablePython",
		"-NoEpicPortal",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected Unreal smoke argv: %#v", args)
	}
	if _, _, err := unrealSmokeInvocation(executable, filepath.Join(t.TempDir(), "caller-project.uproject")); err == nil {
		t.Fatal("Unreal smoke accepted a caller-selected project name")
	}
	if _, _, err := unrealSmokeInvocation(executable, "WorkbenchSmoke.uproject"); err == nil {
		t.Fatal("Unreal smoke accepted a relative project path")
	}
}

func TestUnrealSmokeFailureClassifierReturnsOnlyFixedSafeLabels(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{name: "tnotnull", stderr: `Fatal error: TNotNull<Thing> failed at C:\Users\someone\Secret\file.cpp`, want: "tnotnull-assertion"},
		{name: "assertion", stderr: "Assertion failed: Ptr != nullptr", want: "assertion"},
		{name: "fatal", stderr: "Fatal error: startup failed", want: "fatal"},
		{name: "project", stderr: "Failed to open descriptor file C:\\private\\WorkbenchSmoke.uproject", want: "project-descriptor"},
		{name: "shader", stderr: "Missing global shader FScreenVS", want: "shader-initialization"},
		{name: "zen", stderr: "Zen server connection failed", want: "zen"},
		{name: "quit", stdout: "LogCore: Engine exit requested", want: "quit-observed"},
		{name: "unknown", stderr: "process returned one", want: "nonzero-exit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyUnrealSmokeFailure(tt.stdout, tt.stderr)
			if got != tt.want {
				t.Fatalf("classification=%q want=%q", got, tt.want)
			}
			if strings.Contains(got, `C:\`) || strings.Contains(strings.ToLower(got), "secret") {
				t.Fatalf("classification leaked local detail: %q", got)
			}
		})
	}
}

func TestUnrealSmokeProjectDocumentIsMinimalContentOnlyProject(t *testing.T) {
	if !json.Valid([]byte(unrealSmokeProjectDocument)) {
		t.Fatal("Unreal smoke project document must be valid JSON")
	}
	for _, forbidden := range []string{`"Modules"`, `"Plugins"`} {
		if strings.Contains(unrealSmokeProjectDocument, forbidden) {
			t.Fatalf("Unreal smoke project must not contain %s", forbidden)
		}
	}
}

func TestUnrealGenericHostJobsRemainVersionOnly(t *testing.T) {
	if _, err := validateHostJobSpec(HostJobSpec{Tool: HostBridgeToolUnreal, Operation: HostBridgeOperationVersion}); err != nil {
		t.Fatalf("Unreal version job rejected: %v", err)
	}
	if _, err := validateHostJobSpec(HostJobSpec{Tool: HostBridgeToolUnreal, Operation: HostBridgeOperationUnrealSmoke}); err == nil {
		t.Fatal("generic host submitter accepted Unreal smoke operation")
	}
}

func TestSubmitUnrealSmokeJobUsesDedicatedTypedOperation(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_unrealhost"
	if _, err := RecordHostBridgeHeartbeat(HostBridgeHeartbeat{
		HostID:   hostID,
		Label:    "Unreal test host",
		Platform: HostBridgePlatformWindows,
		Arch:     "amd64",
		Capabilities: map[string]HostCapability{
			HostBridgeToolUnreal: {Installed: true, Version: "Unreal Engine 5.6.1"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	job, err := SubmitUnrealSmokeJob(hostID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Spec.Tool != HostBridgeToolUnreal || job.Spec.Operation != HostBridgeOperationUnrealSmoke || job.Status != "queued" {
		t.Fatalf("unexpected Unreal smoke job: %#v", job)
	}
}

func TestCompareUnrealBuildVersionsUsesNumericComponents(t *testing.T) {
	if compareUnrealBuildVersions(unrealBuildVersion{MajorVersion: 5, MinorVersion: 10}, unrealBuildVersion{MajorVersion: 5, MinorVersion: 9}) <= 0 {
		t.Fatal("5.10 must sort after 5.9")
	}
}
