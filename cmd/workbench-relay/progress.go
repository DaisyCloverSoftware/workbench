package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	relayProgressMinimumLease = 2 * time.Minute
	relayProgressControlGrace = 3 * time.Minute
	relayProgressPublishLease = 15 * time.Minute
)

type relayProgressRecord struct {
	Version      int    `json:"version"`
	PID          int    `json:"pid"`
	Phase        string `json:"phase"`
	UpdatedUnix  int64  `json:"updated_unix"`
	DeadlineUnix int64  `json:"deadline_unix"`
}

var relayProgress struct {
	sync.Mutex
	path      string
	idleLease time.Duration
}

func configureRelayProgress(path string, interval time.Duration) error {
	path = strings.TrimSpace(path)
	idle := relayProgressMinimumLease
	if candidate := interval + time.Minute; candidate > idle {
		idle = candidate
	}
	relayProgress.Lock()
	relayProgress.path = path
	relayProgress.idleLease = idle
	relayProgress.Unlock()
	if path == "" {
		return nil
	}
	return noteRelayProgress("startup", idle)
}

func relayIdleProgressLease() time.Duration {
	relayProgress.Lock()
	defer relayProgress.Unlock()
	if relayProgress.idleLease < relayProgressMinimumLease {
		return relayProgressMinimumLease
	}
	return relayProgress.idleLease
}

func noteRelayProgress(phase string, lease time.Duration) error {
	relayProgress.Lock()
	defer relayProgress.Unlock()
	if relayProgress.path == "" {
		return nil
	}
	if lease < relayProgress.idleLease {
		lease = relayProgress.idleLease
	}
	if lease < relayProgressMinimumLease {
		lease = relayProgressMinimumLease
	}
	now := time.Now().UTC()
	record := relayProgressRecord{
		Version:      1,
		PID:          os.Getpid(),
		Phase:        cleanRelayProgressPhase(phase),
		UpdatedUnix:  now.Unix(),
		DeadlineUnix: now.Add(lease).Unix(),
	}
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(relayProgress.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".workbench-relay-progress-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, relayProgress.path)
}

func cleanRelayProgressPhase(phase string) string {
	phase = strings.ToLower(strings.TrimSpace(phase))
	if phase == "" || len(phase) > 32 {
		return "unknown"
	}
	for _, ch := range phase {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return "unknown"
	}
	return phase
}

func privateControlProgressLease(env privateControlEnvelope) time.Duration {
	lease := relayIdleProgressLease()
	switch env.Action {
	case "run_operations_script":
		var a struct {
			TimeoutSeconds int `json:"timeout_seconds,omitempty"`
		}
		if json.Unmarshal(env.Args, &a) == nil {
			lease = core.OperationsScriptTimeout(a.TimeoutSeconds) + relayProgressControlGrace
		}
	case "inspect_machine", "run_machine_command":
		var a struct {
			TimeoutSeconds int `json:"timeout_seconds,omitempty"`
		}
		if json.Unmarshal(env.Args, &a) == nil {
			lease = core.MachineCommandTimeout(a.TimeoutSeconds) + relayProgressControlGrace
		}
	case "inspect_machine_batch":
		var a struct {
			Commands []struct {
				TimeoutSeconds int `json:"timeout_seconds,omitempty"`
			} `json:"commands"`
		}
		if json.Unmarshal(env.Args, &a) == nil && len(a.Commands) > 0 && len(a.Commands) <= core.MaxMachineInspectBatch {
			lease = relayProgressControlGrace
			for _, command := range a.Commands {
				lease += core.MachineCommandTimeout(command.TimeoutSeconds)
			}
		}
	}
	if lease < relayIdleProgressLease() {
		return relayIdleProgressLease()
	}
	return lease
}
