//go:build windows

package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	windowsHostBridgePollInterval = 10 * time.Second
	windowsCapabilityProbeEvery   = 10 * time.Minute
)

// RunWindowsHostBridgeAgent is an outbound-only local agent. It opens no
// listener. Every network operation is a fixed SSH call to workbench-runner
// host-json, and every local job is revalidated before any executable starts.
func RunWindowsHostBridgeAgent(ctx context.Context, sshHost string) error {
	sshHost, err := validateSSHHostTarget(sshHost)
	if err != nil {
		return err
	}
	hostID, err := loadOrCreateWindowsHostBridgeID()
	if err != nil {
		return err
	}
	label := windowsHostBridgeLabel()

	capabilities := map[string]HostCapability{
		HostBridgeToolBlender: {Installed: false},
		HostBridgeToolUnreal:  {Installed: false},
	}
	var nextCapabilityProbe time.Time
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		now := time.Now()
		if nextCapabilityProbe.IsZero() || !now.Before(nextCapabilityProbe) {
			probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			capabilities = discoverWindowsHostCapabilities(probeCtx)
			cancel()
			nextCapabilityProbe = now.Add(windowsCapabilityProbeEvery)
		}

		heartbeat := HostBridgeHeartbeat{
			HostID:       hostID,
			Label:        label,
			Platform:     HostBridgePlatformWindows,
			Arch:         runtime.GOARCH,
			Capabilities: capabilities,
		}
		pollCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		response, pollErr := RunHostBridgeRPCSSH(pollCtx, sshHost, HostBridgeRPCRequest{
			Action:    HostBridgeRPCPoll,
			Heartbeat: &heartbeat,
		})
		cancel()
		if pollErr == nil && response.Job != nil {
			result, jobErr := executeWindowsHostBridgeJob(ctx, hostID, *response.Job)
			completeCtx, completeCancel := context.WithTimeout(ctx, 20*time.Second)
			_, _ = RunHostBridgeRPCSSH(completeCtx, sshHost, HostBridgeRPCRequest{
				Action: HostBridgeRPCComplete,
				HostID: hostID,
				JobID:  response.Job.ID,
				Result: &result,
				Error:  jobErr,
			})
			completeCancel()
			if jobErr == "" && response.Job.Spec.Operation == HostBridgeOperationVersion {
				switch response.Job.Spec.Tool {
				case HostBridgeToolBlender:
					capabilities[HostBridgeToolBlender] = HostCapability{Installed: true, Version: result.Output}
				case HostBridgeToolUnreal:
					capabilities[HostBridgeToolUnreal] = HostCapability{Installed: true, Version: result.Output}
				}
			}
		}

		timer := time.NewTimer(windowsHostBridgePollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func executeWindowsHostBridgeJob(ctx context.Context, hostID string, job HostJob) (HostJobResult, string) {
	if job.HostID != hostID || job.ClaimedBy != hostID || job.Status != "claimed" {
		return HostJobResult{ExitCode: 1}, "Windows host bridge rejected a job that was not claimed by this host"
	}

	switch job.Spec.Tool {
	case HostBridgeToolBlender:
		executable := findBlenderExecutable()
		if executable == "" {
			return HostJobResult{ExitCode: 1}, "Blender is not installed in an allowlisted Windows location"
		}
		switch job.Spec.Operation {
		case HostBridgeOperationVersion:
			probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			version, err := runBlenderVersion(probeCtx, executable)
			if err != nil {
				return HostJobResult{ExitCode: 1}, err.Error()
			}
			return HostJobResult{Output: version, ExitCode: 0}, ""
		case HostBridgeOperationBlenderSmokeRender:
			output, err := runBlenderSmokeRender(ctx, executable, job.ID)
			if err != nil {
				return HostJobResult{ExitCode: 1}, err.Error()
			}
			return HostJobResult{Output: output, ExitCode: 0}, ""
		default:
			return HostJobResult{ExitCode: 1}, "Windows host bridge rejected an unsupported Blender operation"
		}

	case HostBridgeToolUnreal:
		executable := findUnrealEditorCmdExecutable()
		if executable == "" {
			return HostJobResult{ExitCode: 1}, "Unreal Editor is not installed in an allowlisted Windows location"
		}
		switch job.Spec.Operation {
		case HostBridgeOperationVersion:
			version, err := runUnrealVersion(executable)
			if err != nil {
				return HostJobResult{ExitCode: 1}, err.Error()
			}
			return HostJobResult{Output: version, ExitCode: 0}, ""
		case HostBridgeOperationUnrealSmoke:
			output, err := runUnrealSmoke(ctx, executable)
			if err != nil {
				return HostJobResult{ExitCode: 1}, err.Error()
			}
			return HostJobResult{Output: output, ExitCode: 0}, ""
		default:
			return HostJobResult{ExitCode: 1}, "Windows host bridge rejected an unsupported Unreal operation"
		}

	default:
		return HostJobResult{ExitCode: 1}, "Windows host bridge rejected an unsupported local tool"
	}
}

func discoverWindowsHostCapabilities(ctx context.Context) map[string]HostCapability {
	capabilities := map[string]HostCapability{
		HostBridgeToolBlender: {Installed: false},
		HostBridgeToolUnreal:  {Installed: false},
	}
	if executable := findBlenderExecutable(); executable != "" {
		if version, err := runBlenderVersion(ctx, executable); err == nil {
			capabilities[HostBridgeToolBlender] = HostCapability{Installed: true, Version: version}
		}
	}
	if executable := findUnrealEditorCmdExecutable(); executable != "" {
		if version, err := runUnrealVersion(executable); err == nil {
			capabilities[HostBridgeToolUnreal] = HostCapability{Installed: true, Version: version}
		}
	}
	return capabilities
}

func runBlenderVersion(ctx context.Context, executable string) (string, error) {
	name, args, err := blenderVersionInvocation(executable)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(8 << 10)
	stderr := newBoundedWorkerCapture(8 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if runErr != nil {
		if ctx.Err() != nil {
			return "", errors.New("Blender version probe timed out")
		}
		return "", errors.New("Blender version probe failed")
	}
	return blenderVersionFromCapturedOutput(stdout, stderr)
}

func findBlenderExecutable() string {
	var candidates []string
	seen := map[string]bool{}
	for _, root := range []string{os.Getenv("ProgramW6432"), os.Getenv("ProgramFiles")} {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, "Blender Foundation", "Blender *", "blender.exe"))
		for _, candidate := range matches {
			candidate = filepath.Clean(candidate)
			if !seen[strings.ToLower(candidate)] {
				seen[strings.ToLower(candidate)] = true
				candidates = append(candidates, candidate)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			if _, _, err := blenderVersionInvocation(candidate); err == nil {
				return candidate
			}
		}
	}
	if candidate, err := exec.LookPath("blender.exe"); err == nil {
		candidate = filepath.Clean(candidate)
		if _, _, err := blenderVersionInvocation(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func loadOrCreateWindowsHostBridgeID() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "windows-host-id")
	if data, readErr := os.ReadFile(path); readErr == nil {
		return validateHostBridgeID(string(data))
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return "", readErr
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := "windows_" + hex.EncodeToString(b)
	if _, err := validateHostBridgeID(id); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".windows-host-id-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if _, err := tmp.WriteString(id + "\n"); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", err
	}
	return id, nil
}

func windowsHostBridgeLabel() string {
	label, _ := os.Hostname()
	label = strings.TrimSpace(label)
	if label == "" {
		label = "Windows workstation"
	}
	label = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '-'
		}
		return r
	}, label)
	if len(label) > 100 {
		label = label[:100]
	}
	if LooksSecret(label) {
		return "Windows workstation"
	}
	return label
}
