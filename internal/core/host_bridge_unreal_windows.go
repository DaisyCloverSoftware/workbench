//go:build windows

package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type unrealInstallCandidate struct {
	Executable string
	Version    unrealBuildVersion
}

type epicLauncherInstallManifest struct {
	InstallationList []struct {
		InstallLocation string `json:"InstallLocation"`
		AppName         string `json:"AppName"`
	} `json:"InstallationList"`
}

func findUnrealEditorCmdExecutable() string {
	seen := map[string]bool{}
	var paths []string
	add := func(candidate string) {
		candidate = filepath.Clean(strings.TrimSpace(candidate))
		if candidate == "." || candidate == "" {
			return
		}
		key := strings.ToLower(candidate)
		if seen[key] {
			return
		}
		seen[key] = true
		paths = append(paths, candidate)
	}

	for _, installRoot := range epicLauncherUnrealInstallRoots() {
		add(filepath.Join(installRoot, "Engine", "Binaries", "Win64", "UnrealEditor-Cmd.exe"))
	}
	for _, root := range []string{os.Getenv("ProgramW6432"), os.Getenv("ProgramFiles")} {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(root, "Epic Games", "UE_*", "Engine", "Binaries", "Win64", "UnrealEditor-Cmd.exe"))
		for _, candidate := range matches {
			add(candidate)
		}
	}
	if candidate, err := exec.LookPath("UnrealEditor-Cmd.exe"); err == nil {
		if abs, absErr := filepath.Abs(candidate); absErr == nil {
			add(abs)
		}
	}

	candidates := make([]unrealInstallCandidate, 0, len(paths))
	for _, executable := range paths {
		info, err := os.Lstat(executable)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			continue
		}
		version, err := readUnrealBuildVersion(executable)
		if err != nil {
			continue
		}
		candidates = append(candidates, unrealInstallCandidate{Executable: executable, Version: version})
	}
	sort.Slice(candidates, func(i, j int) bool {
		cmp := compareUnrealBuildVersions(candidates[i].Version, candidates[j].Version)
		if cmp != 0 {
			return cmp > 0
		}
		return strings.ToLower(candidates[i].Executable) < strings.ToLower(candidates[j].Executable)
	})
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].Executable
}

func epicLauncherUnrealInstallRoots() []string {
	programData := strings.TrimSpace(os.Getenv("ProgramData"))
	if programData == "" {
		return nil
	}
	path := filepath.Join(programData, "Epic", "UnrealEngineLauncher", "LauncherInstalled.dat")
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4<<20 {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var manifest epicLauncherInstallManifest
	if json.Unmarshal(raw, &manifest) != nil {
		return nil
	}
	seen := map[string]bool{}
	var roots []string
	for _, install := range manifest.InstallationList {
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(install.AppName)), "UE_") {
			continue
		}
		root := filepath.Clean(strings.TrimSpace(install.InstallLocation))
		if root == "." || root == "" || !filepath.IsAbs(root) {
			continue
		}
		key := strings.ToLower(root)
		if seen[key] {
			continue
		}
		seen[key] = true
		roots = append(roots, root)
	}
	return roots
}

func runUnrealVersion(executable string) (string, error) {
	version, err := readUnrealBuildVersion(executable)
	if err != nil {
		return "", err
	}
	return formatUnrealVersion(version), nil
}

func runUnrealSmoke(ctx context.Context, executable string) (string, error) {
	name, args, err := unrealSmokeInvocation(executable)
	if err != nil {
		return "", err
	}
	version, err := runUnrealVersion(executable)
	if err != nil {
		return "", err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, name, args...)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(8 << 10)
	stderr := newBoundedWorkerCapture(8 << 10)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if probeCtx.Err() != nil {
			return "", errors.New("Unreal headless smoke timed out")
		}
		return "", errors.New("Unreal headless smoke failed")
	}
	return "Unreal headless smoke complete: " + version, nil
}
