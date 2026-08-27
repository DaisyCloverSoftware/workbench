package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRelayProgressWritesCurrentPIDAndBoundedLease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "relay-progress.json")
	if err := configureRelayProgress(path, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		relayProgress.Lock()
		relayProgress.path = ""
		relayProgress.idleLease = 0
		relayProgress.Unlock()
	})
	before := time.Now().Unix()
	if err := noteRelayProgress("poll", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record relayProgressRecord
	if err := json.Unmarshal(b, &record); err != nil {
		t.Fatal(err)
	}
	if record.Version != 1 || record.PID != os.Getpid() || record.Phase != "poll" {
		t.Fatalf("unexpected progress record: %+v", record)
	}
	if record.UpdatedUnix < before || record.DeadlineUnix-record.UpdatedUnix < int64(relayProgressMinimumLease/time.Second) {
		t.Fatalf("progress lease was not bounded: %+v", record)
	}
}

func TestPrivateControlProgressLeaseTracksDeclaredTimeouts(t *testing.T) {
	if err := configureRelayProgress("", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	operation := privateControlEnvelope{Action: "run_operations_script", Args: json.RawMessage(`{"timeout_seconds":90}`)}
	if got, want := privateControlProgressLease(operation), 90*time.Second+relayProgressControlGrace; got != want {
		t.Fatalf("operation lease=%s want=%s", got, want)
	}

	machine := privateControlEnvelope{Action: "inspect_machine", Args: json.RawMessage(`{"timeout_seconds":600}`)}
	if got, want := privateControlProgressLease(machine), 10*time.Minute+relayProgressControlGrace; got != want {
		t.Fatalf("machine lease=%s want=%s", got, want)
	}

	batch := privateControlEnvelope{Action: "inspect_machine_batch", Args: json.RawMessage(`{"commands":[{"timeout_seconds":600},{"timeout_seconds":90}]}`)}
	if got, want := privateControlProgressLease(batch), 10*time.Minute+90*time.Second+relayProgressControlGrace; got != want {
		t.Fatalf("batch lease=%s want=%s", got, want)
	}
}

func TestRelayProgressPhaseRejectsUntrustedText(t *testing.T) {
	if got := cleanRelayProgressPhase("control-execute"); got != "control-execute" {
		t.Fatalf("clean phase=%q", got)
	}
	for _, bad := range []string{"control execute", "x/y", "$(touch nope)", ""} {
		if got := cleanRelayProgressPhase(bad); got != "unknown" {
			t.Fatalf("unsafe phase %q became %q", bad, got)
		}
	}
}
