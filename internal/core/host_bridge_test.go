package core

import (
	"testing"
)

func TestHostBridgeQueueLifecycle(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_testhost"
	host, err := RecordHostBridgeHeartbeat(HostBridgeHeartbeat{
		HostID:   hostID,
		Label:    "Test Windows host",
		Platform: HostBridgePlatformWindows,
		Arch:     "amd64",
		Capabilities: map[string]HostCapability{
			HostBridgeToolBlender: {Installed: true, Version: "Blender 4.5.0"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !host.Online || host.HostID != hostID {
		t.Fatalf("unexpected host: %#v", host)
	}

	job, err := SubmitHostBridgeJob(hostID, HostJobSpec{Tool: HostBridgeToolBlender, Operation: HostBridgeOperationVersion})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := ClaimHostBridgeJob(hostID)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != job.ID || claimed.Status != "claimed" || claimed.ClaimedBy != hostID {
		t.Fatalf("unexpected claimed job: %#v", claimed)
	}
	completed, err := CompleteHostBridgeJob(hostID, job.ID, HostJobResult{Output: "Blender 4.5.0", ExitCode: 0}, "")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || completed.Result == nil || completed.Result.Output != "Blender 4.5.0" {
		t.Fatalf("unexpected completion: %#v", completed)
	}
}

func TestHostBridgeRPCPollClaimsAtMostTypedVersionJob(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	hostID := "windows_rpchost"
	heartbeat := HostBridgeHeartbeat{HostID: hostID, Label: "RPC Windows host", Platform: HostBridgePlatformWindows, Arch: "amd64"}
	if _, err := RecordHostBridgeHeartbeat(heartbeat); err != nil {
		t.Fatal(err)
	}
	if _, err := SubmitHostBridgeJob(hostID, HostJobSpec{Tool: HostBridgeToolBlender, Operation: HostBridgeOperationVersion}); err != nil {
		t.Fatal(err)
	}

	response, err := ApplyHostBridgeRPC(HostBridgeRPCRequest{Action: HostBridgeRPCPoll, Heartbeat: &heartbeat})
	if err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Job == nil || response.Job.Spec.Tool != HostBridgeToolBlender || response.Job.Spec.Operation != HostBridgeOperationVersion {
		t.Fatalf("unexpected poll response: %#v", response)
	}
	if _, err := ApplyHostBridgeRPC(HostBridgeRPCRequest{Action: HostBridgeRPCPoll, Heartbeat: &heartbeat, JobID: response.Job.ID}); err == nil {
		t.Fatal("poll accepted completion-only fields")
	}
}

func TestHostBridgeRejectsArbitraryOperations(t *testing.T) {
	for _, spec := range []HostJobSpec{
		{Tool: "powershell", Operation: "version"},
		{Tool: HostBridgeToolBlender, Operation: "render"},
		{Tool: HostBridgeToolBlender, Operation: "command"},
	} {
		if _, err := validateHostJobSpec(spec); err == nil {
			t.Fatalf("unsafe host job was accepted: %#v", spec)
		}
	}
}

func TestBlenderVersionInvocationIsExactArgv(t *testing.T) {
	name, args, err := blenderVersionInvocation("blender.exe")
	if err != nil {
		t.Fatal(err)
	}
	if name != "blender.exe" || len(args) != 1 || args[0] != "--version" {
		t.Fatalf("unexpected invocation: name=%q args=%#v", name, args)
	}
	if _, _, err := blenderVersionInvocation("cmd.exe"); err == nil {
		t.Fatal("non-Blender executable was accepted")
	}
	if _, _, err := blenderVersionInvocation("blender.exe --background"); err == nil {
		t.Fatal("command-like Blender executable string was accepted")
	}
}

func TestParseBlenderVersionOutput(t *testing.T) {
	version, err := parseBlenderVersionOutput("Blender 4.5.0 LTS\n")
	if err != nil {
		t.Fatal(err)
	}
	if version != "Blender 4.5.0 LTS" {
		t.Fatalf("version=%q", version)
	}
	if _, err := parseBlenderVersionOutput("not blender\n"); err == nil {
		t.Fatal("unexpected version output was accepted")
	}
}
