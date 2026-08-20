package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlenderSmokeRenderInvocationIsFixedArgv(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "smoke_")
	name, args, err := blenderSmokeRenderInvocation("blender.exe", prefix)
	if err != nil {
		t.Fatal(err)
	}
	if name != "blender.exe" {
		t.Fatalf("unexpected executable %q", name)
	}
	want := []string{
		"--background",
		"--factory-startup",
		"--disable-autoexec",
		"--render-output", prefix,
		"--render-format", "PNG",
		"--render-frame", "1",
	}
	if len(args) != len(want) {
		t.Fatalf("unexpected args %#v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg %d=%q want %q; all=%#v", i, args[i], want[i], args)
		}
	}
	if _, _, err := blenderSmokeRenderInvocation("cmd.exe", prefix); err == nil {
		t.Fatal("non-Blender executable was accepted")
	}
	if _, _, err := blenderSmokeRenderInvocation("blender.exe", filepath.Join(t.TempDir(), "other")); err == nil {
		t.Fatal("non-fixed output basename was accepted")
	}
}

func TestDedicatedBlenderSmokeRenderSubmitDoesNotWidenGenericHostJobs(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_renderhost"
	if _, err := RecordHostBridgeHeartbeat(HostBridgeHeartbeat{
		HostID:   hostID,
		Label:    "Render Windows host",
		Platform: HostBridgePlatformWindows,
		Arch:     "amd64",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := SubmitHostBridgeJob(hostID, HostJobSpec{Tool: HostBridgeToolBlender, Operation: HostBridgeOperationBlenderSmokeRender}); err == nil {
		t.Fatal("generic host-job submitter unexpectedly accepted render_smoke")
	}
	job, err := SubmitBlenderSmokeRenderJob(hostID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Spec.Tool != HostBridgeToolBlender || job.Spec.Operation != HostBridgeOperationBlenderSmokeRender || job.Status != "queued" {
		t.Fatalf("unexpected smoke render job %#v", job)
	}
}

func TestVerifyBlenderSmokeRenderReturnsOnlyBoundedMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "smoke_0001.png")
	payload := append(append([]byte{}, blenderSmokePNGSignature...), []byte("bounded-smoke-render")...)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := verifyBlenderSmokeRender(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Blender smoke render complete", "artifact=smoke_0001.png", "bytes=", "sha256="} {
		if !strings.Contains(got, want) {
			t.Fatalf("metadata missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, filepath.Dir(path)) {
		t.Fatalf("local render path leaked into host result: %q", got)
	}
}

func TestVerifyBlenderSmokeRenderRejectsNonPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "smoke_0001.png")
	if err := os.WriteFile(path, []byte("not a png"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyBlenderSmokeRender(path); err == nil {
		t.Fatal("non-PNG smoke render output was accepted")
	}
}

func TestBlenderSmokeRenderOperationsScriptStaysNarrow(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "scripts", "ops", "windows-blender-smoke-render.sh"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(b)
	for _, want := range []string{
		"set -euo pipefail",
		"^windows_[a-z0-9_-]{8,95}$",
		"run ./cmd/workbench-blender-smoke-submit",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("operations script missing %q", want)
		}
	}
	for _, forbidden := range []string{"eval ", "bash -c", "powershell", "cmd.exe", "--python", "--python-expr"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("operations script contains forbidden command surface %q", forbidden)
		}
	}
}
