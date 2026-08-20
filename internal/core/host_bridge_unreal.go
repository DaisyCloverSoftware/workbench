package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const HostBridgeOperationUnrealSmoke = "smoke"

const unrealSmokeProjectDocument = `{
  "FileVersion": 3,
  "EngineAssociation": "",
  "Category": "",
  "Description": "Workbench bounded Unreal smoke project"
}
`

type unrealBuildVersion struct {
	MajorVersion int `json:"MajorVersion"`
	MinorVersion int `json:"MinorVersion"`
	PatchVersion int `json:"PatchVersion"`
	Changelist   int `json:"Changelist"`
}

func unrealVersionFileForExecutable(executable string) (string, error) {
	executable = filepath.Clean(strings.TrimSpace(executable))
	if executable == "" || strings.ContainsAny(executable, "\r\n\x00") {
		return "", errors.New("Unreal executable path is invalid")
	}
	if !filepath.IsAbs(executable) {
		return "", errors.New("Unreal executable path must be absolute")
	}
	if !strings.EqualFold(filepath.Base(executable), "UnrealEditor-Cmd.exe") {
		return "", errors.New("Unreal executable must be UnrealEditor-Cmd.exe")
	}
	win64 := filepath.Dir(executable)
	binaries := filepath.Dir(win64)
	engine := filepath.Dir(binaries)
	if !strings.EqualFold(filepath.Base(win64), "Win64") ||
		!strings.EqualFold(filepath.Base(binaries), "Binaries") ||
		!strings.EqualFold(filepath.Base(engine), "Engine") {
		return "", errors.New("Unreal executable is outside the expected Engine/Binaries/Win64 layout")
	}
	return filepath.Join(engine, "Build", "Build.version"), nil
}

func readUnrealBuildVersion(executable string) (unrealBuildVersion, error) {
	versionFile, err := unrealVersionFileForExecutable(executable)
	if err != nil {
		return unrealBuildVersion{}, err
	}
	info, err := os.Lstat(versionFile)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<10 {
		return unrealBuildVersion{}, errors.New("Unreal Build.version is missing or invalid")
	}
	raw, err := os.ReadFile(versionFile)
	if err != nil {
		return unrealBuildVersion{}, errors.New("Unreal Build.version could not be read")
	}
	var version unrealBuildVersion
	if err := json.Unmarshal(raw, &version); err != nil {
		return unrealBuildVersion{}, errors.New("Unreal Build.version is malformed")
	}
	if version.MajorVersion < 4 || version.MajorVersion > 99 || version.MinorVersion < 0 || version.MinorVersion > 99 || version.PatchVersion < 0 || version.PatchVersion > 999 {
		return unrealBuildVersion{}, errors.New("Unreal Build.version contains an invalid engine version")
	}
	return version, nil
}

func formatUnrealVersion(version unrealBuildVersion) string {
	return fmt.Sprintf("Unreal Engine %d.%d.%d", version.MajorVersion, version.MinorVersion, version.PatchVersion)
}

func compareUnrealBuildVersions(a, b unrealBuildVersion) int {
	if a.MajorVersion != b.MajorVersion {
		if a.MajorVersion < b.MajorVersion {
			return -1
		}
		return 1
	}
	if a.MinorVersion != b.MinorVersion {
		if a.MinorVersion < b.MinorVersion {
			return -1
		}
		return 1
	}
	if a.PatchVersion != b.PatchVersion {
		if a.PatchVersion < b.PatchVersion {
			return -1
		}
		return 1
	}
	if a.Changelist != b.Changelist {
		if a.Changelist < b.Changelist {
			return -1
		}
		return 1
	}
	return 0
}

func unrealSmokeInvocation(executable, project string) (string, []string, error) {
	if _, err := unrealVersionFileForExecutable(executable); err != nil {
		return "", nil, err
	}
	project = filepath.Clean(strings.TrimSpace(project))
	if project == "" || project == "." || strings.ContainsAny(project, "\r\n\x00") {
		return "", nil, errors.New("Unreal smoke project path is invalid")
	}
	if !filepath.IsAbs(project) {
		return "", nil, errors.New("Unreal smoke project path must be absolute")
	}
	if !strings.EqualFold(filepath.Base(project), "WorkbenchSmoke.uproject") {
		return "", nil, errors.New("Unreal smoke project must use the fixed WorkbenchSmoke.uproject name")
	}
	info, err := os.Lstat(project)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<10 {
		return "", nil, errors.New("Unreal smoke project is missing or invalid")
	}
	return filepath.Clean(executable), []string{
		project,
		"-ExecCmds=Quit",
		"-unattended",
		"-stdout",
		"-nop4",
		"-nosplash",
		"-DisablePython",
		"-NoEpicPortal",
		"-nocrashreports",
	}, nil
}

// classifyUnrealSmokeFailure deliberately maps bounded local process output to a
// tiny fixed vocabulary. Raw Unreal stdout/stderr can contain local paths and
// machine details, so it must never cross the Windows host bridge merely to
// diagnose a startup failure.
func classifyUnrealSmokeFailure(stdout, stderr string) string {
	text := strings.ToLower(stdout + "\n" + stderr)
	switch {
	case strings.Contains(text, "tnotnull"):
		return "tnotnull-assertion"
	case strings.Contains(text, "assertion failed") || strings.Contains(text, "assert failed"):
		return "assertion"
	case strings.Contains(text, "fatal error") || strings.Contains(text, "app error called"):
		return "fatal"
	case strings.Contains(text, "failed to open descriptor file") || strings.Contains(text, "project file not found"):
		return "project-descriptor"
	case strings.Contains(text, "missing global shader") || strings.Contains(text, "failed to compile global shader"):
		return "shader-initialization"
	case strings.Contains(text, "zen") && (strings.Contains(text, "failed") || strings.Contains(text, "unable") || strings.Contains(text, "error")):
		return "zen"
	case strings.Contains(text, "engine exit requested") || strings.Contains(text, "requestengineexit"):
		return "quit-observed"
	default:
		return "nonzero-exit"
	}
}

// SubmitUnrealSmokeJob is deliberately separate from the generic host-job
// submitter. The generic path remains version-only. This function creates one
// fixed UnrealEditor-Cmd startup-and-quit smoke. The Windows agent itself creates
// the disposable project; callers cannot supply a project, script, commandlet,
// executable, path or arguments.
func SubmitUnrealSmokeJob(hostID string) (HostJob, error) {
	hostID, err := validateHostBridgeID(hostID)
	if err != nil {
		return HostJob{}, err
	}
	jobID, err := newHostJobID()
	if err != nil {
		return HostJob{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	job := HostJob{
		ID:        jobID,
		HostID:    hostID,
		Spec:      HostJobSpec{Tool: HostBridgeToolUnreal, Operation: HostBridgeOperationUnrealSmoke},
		Status:    "queued",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = withHostBridgeLock(func(root string) error {
		var host HostBridgeHost
		if err := readHostBridgeJSON(filepath.Join(root, "hosts", hostID+".json"), &host); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return errors.New("target host is not registered")
			}
			return err
		}
		return writeHostBridgeJSON(filepath.Join(root, "jobs", job.ID+".json"), job)
	})
	return job, err
}
