package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxRunnerHarnessConfigBytes = 64 << 10

var (
	runnerHarnessConfigMu           sync.RWMutex
	runnerHarnessConfigPathOverride string
)

type RunnerHarnessConfig struct {
	Version     int       `json:"version"`
	AdapterPath string    `json:"adapter_path"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RunnerHarnessStatus struct {
	Configured  bool   `json:"configured"`
	Available   bool   `json:"available"`
	AdapterName string `json:"adapter_name,omitempty"`
	Status      string `json:"status"`
}

func RunnerHarnessConfigPath() (string, error) {
	if override := strings.TrimSpace(runnerHarnessConfigPathOverride); override != "" {
		path, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return "", err
		}
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "runner-harness.json"), nil
}

func LoadRunnerHarnessConfig() (RunnerHarnessConfig, bool, error) {
	runnerHarnessConfigMu.RLock()
	defer runnerHarnessConfigMu.RUnlock()
	return loadRunnerHarnessConfigUnlocked()
}

func SaveRunnerHarnessAdapter(path string) (RunnerHarnessConfig, error) {
	resolved, err := validateHarnessAdapterPath(path)
	if err != nil {
		return RunnerHarnessConfig{}, err
	}
	cfg := RunnerHarnessConfig{Version: 1, AdapterPath: resolved, UpdatedAt: time.Now().UTC()}
	release, err := lockRunnerHarnessConfigWrite()
	if err != nil {
		return RunnerHarnessConfig{}, err
	}
	defer release()
	if err := saveRunnerHarnessConfigUnlocked(cfg); err != nil {
		return RunnerHarnessConfig{}, err
	}
	return cfg, nil
}

func DeleteRunnerHarnessAdapter() error {
	release, err := lockRunnerHarnessConfigWrite()
	if err != nil {
		return err
	}
	defer release()
	path, err := RunnerHarnessConfigPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func RunnerHarnessConfigurationStatus() RunnerHarnessStatus {
	cfg, configured, err := LoadRunnerHarnessConfig()
	if err != nil {
		return RunnerHarnessStatus{Status: "configuration unavailable"}
	}
	if !configured {
		return RunnerHarnessStatus{Status: "not configured"}
	}
	status := RunnerHarnessStatus{Configured: true, AdapterName: filepath.Base(cfg.AdapterPath)}
	if _, err := validateHarnessAdapterPath(cfg.AdapterPath); err != nil {
		status.Status = "configured adapter executable is unavailable"
		return status
	}
	status.Available = true
	status.Status = "configured · structured protocol v1"
	return status
}

func loadRunnerHarnessConfigUnlocked() (RunnerHarnessConfig, bool, error) {
	path, err := RunnerHarnessConfigPath()
	if err != nil {
		return RunnerHarnessConfig{}, false, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return RunnerHarnessConfig{}, false, nil
	}
	if err != nil {
		return RunnerHarnessConfig{}, false, err
	}
	if !info.Mode().IsRegular() {
		return RunnerHarnessConfig{}, false, errors.New("runner harness configuration is not a regular file")
	}
	if info.Size() > maxRunnerHarnessConfigBytes {
		return RunnerHarnessConfig{}, false, errors.New("runner harness configuration is unexpectedly large")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return RunnerHarnessConfig{}, false, err
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	var cfg RunnerHarnessConfig
	if err := dec.Decode(&cfg); err != nil {
		return RunnerHarnessConfig{}, false, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return RunnerHarnessConfig{}, false, errors.New("runner harness configuration contains more than one JSON value")
		}
		return RunnerHarnessConfig{}, false, err
	}
	if cfg.Version != 1 {
		return RunnerHarnessConfig{}, false, fmt.Errorf("unsupported runner harness configuration version %d", cfg.Version)
	}
	cfg.AdapterPath = strings.TrimSpace(cfg.AdapterPath)
	if cfg.AdapterPath == "" {
		return RunnerHarnessConfig{}, false, errors.New("runner harness configuration has no adapter path")
	}
	return cfg, true, nil
}

func saveRunnerHarnessConfigUnlocked(cfg RunnerHarnessConfig) error {
	path, err := RunnerHarnessConfigPath()
	if err != nil {
		return err
	}
	cfg.Version = 1
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if len(body) > maxRunnerHarnessConfigBytes {
		return errors.New("runner harness configuration exceeds its local size limit")
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".runner-harness-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func lockRunnerHarnessConfigWrite() (func(), error) {
	runnerHarnessConfigMu.Lock()
	path, err := RunnerHarnessConfigPath()
	if err != nil {
		runnerHarnessConfigMu.Unlock()
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, openErr := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() {
				_ = os.Remove(lockPath)
				runnerHarnessConfigMu.Unlock()
			}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			runnerHarnessConfigMu.Unlock()
			return nil, openErr
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 2*time.Minute {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			runnerHarnessConfigMu.Unlock()
			return nil, errors.New("timed out waiting for Workbench runner-harness configuration lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
