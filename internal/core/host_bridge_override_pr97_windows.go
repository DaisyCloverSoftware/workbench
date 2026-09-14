//go:build windows

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Execute PR97's existing reviewed scripts, not a substitute recipe. Callers
// cannot choose a program, argv, script, ref, engine or directory.
func runOverridePR97Proof(ctx context.Context, jobID string) (output string, resultErr error) {
	id, err := validateHostBridgeID(jobID)
	if err != nil || id != jobID || !strings.HasPrefix(id, "hostjob_") {
		return "", errors.New("PR97 job id is invalid")
	}
	cache, err := os.UserCacheDir()
	if err != nil || !filepath.IsAbs(cache) {
		return "", errors.New("Workbench cache unavailable")
	}
	base := filepath.Join(cache, "Workbench", "override-pr97-proof")
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	if err := overridePR97NoAlias(base); err != nil {
		return "", err
	}
	// A crashed build leaves a lock rather than allowing concurrent writers.
	lock := filepath.Join(base, "active")
	if err := os.Mkdir(lock, 0700); err != nil {
		return "", errors.New("PR97 build already active or requires recovery; no workspace was changed")
	}
	defer os.Remove(lock)
	work := filepath.Join(base, id)
	if err := os.Mkdir(work, 0700); err != nil {
		return "", errors.New("PR97 job workspace already exists or is unavailable")
	}
	project := filepath.Join(work, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		return "", err
	}
	log, err := os.OpenFile(filepath.Join(work, "operation.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	defer log.Close()
	stage := "preflight"
	completed := false
	evidence := map[string]any{"schema_version": 1, "job_id": id, "source_sha": overridePR97SourceSHA, "repository": overridePR97Repository, "map": overridePR97Map, "started_utc": time.Now().UTC().Format(time.RFC3339Nano), "visual_acceptance": "not_assessed", "packaged_runtime_proved": false}
	defer func() {
		evidence["stage"] = stage
		evidence["success"] = completed
		evidence["finished_utc"] = time.Now().UTC().Format(time.RFC3339Nano)
		if resultErr != nil {
			evidence["error"] = resultErr.Error()
		}
		raw, marshalErr := json.Marshal(evidence)
		if marshalErr == nil {
			writeErr := os.WriteFile(filepath.Join(work, "operation.json"), raw, 0600)
			if writeErr != nil && resultErr == nil {
				resultErr = errors.New("PR97 proof manifest could not be preserved")
				evidence["success"] = false
				evidence["error"] = resultErr.Error()
				raw, _ = json.Marshal(evidence)
			}
			output = string(raw)
		} else if resultErr == nil {
			resultErr = marshalErr
		}
	}()
	programs := strings.TrimSpace(os.Getenv("ProgramW6432"))
	if programs == "" {
		programs = strings.TrimSpace(os.Getenv("ProgramFiles"))
	}
	if !filepath.IsAbs(programs) {
		return "", errors.New("Windows Program Files root unavailable")
	}
	git := filepath.Join(programs, "Git", "cmd", "git.exe")
	pwsh := filepath.Join(programs, "PowerShell", "7", "pwsh.exe")
	for _, file := range []string{git, pwsh} {
		if err := overridePR97Regular(file); err != nil {
			return "", err
		}
	}
	// Fixed checked-in project association, not caller-provided UE_ROOT or PATH.
	engine, err := resolveOverrideRinAssociatedEngine(ctx, overrideRinEngineAssociation)
	if err != nil {
		return "", err
	}
	if err := overridePR97NoAlias(engine); err != nil {
		return "", err
	}
	editor := filepath.Join(engine, "Engine", "Binaries", "Win64", "UnrealEditor-Cmd.exe")
	python := filepath.Join(engine, "Engine", "Binaries", "ThirdParty", "Python3", "Win64", "python.exe")
	for _, file := range []string{editor, python, filepath.Join(engine, "Engine", "Build", "BatchFiles", "Build.bat"), filepath.Join(engine, "Engine", "Build", "BatchFiles", "RunUAT.bat")} {
		if err := overridePR97Regular(file); err != nil {
			return "", err
		}
	}
	version, err := runUnrealVersion(editor)
	if err != nil || version != "Unreal Engine 5.8.1" {
		return "", errors.New("PR97 requires the associated Unreal Engine 5.8.1 installation")
	}
	evidence["engine_version"] = version
	engineHash, err := sha256RegularFile(filepath.Join(engine, "Engine", "Build", "Build.version"))
	if err != nil {
		return "", err
	}
	evidence["engine_build_version_sha256"] = engineHash
	env := overrideRinGitEnvironment()
	env = append(env, "PATH="+filepath.Dir(git)+string(os.PathListSeparator)+filepath.Dir(python)+string(os.PathListSeparator)+os.Getenv("PATH"))
	run := func(program string, args ...string) error {
		fmt.Fprintln(log, "STAGE", stage)
		if err := overridePR97RunProcess(ctx, project, env, log, program, args...); err != nil {
			return fmt.Errorf("PR97 %s failed; retained operation.log and Saved/Validation: %w", stage, err)
		}
		return nil
	}
	stage = "exact-source"
	for _, args := range [][]string{{"init", "."}, {"remote", "add", "origin", overridePR97Repository}, {"config", "core.autocrlf", "false"}, {"config", "advice.detachedHead", "false"}, {"fetch", "--no-tags", "--depth=1", "origin", overridePR97SourceSHA}, {"checkout", "--detach", overridePR97SourceSHA}} {
		if err := run(git, args...); err != nil {
			return "", err
		}
	}
	if err := overridePR97VerifyCheckout(ctx, git, project); err != nil {
		return "", err
	}
	projectFile := filepath.Join(project, overrideRinProjectName)
	if err := overridePR97Regular(projectFile); err != nil {
		return "", err
	}
	if err := validateOverrideRinProject(projectFile); err != nil {
		return "", err
	}
	recipes := map[string]string{"tools/windows/validate_real_unreal.ps1": "ea346bd44ee1684fed85f247a02ba623c17a0b42", "tools/windows/build_win64_review.ps1": "188ed4cc1a973406799302d4c11dd0a78026fdc2"}
	for path, expected := range recipes {
		file := filepath.Join(project, filepath.FromSlash(path))
		if err := overridePR97Regular(file); err != nil {
			return "", err
		}
		actual, err := outputOverrideRinGit(ctx, git, project, "hash-object", "--no-filters", path)
		if err != nil || strings.TrimSpace(string(actual)) != expected {
			return "", errors.New("PR97 fixed recipe integrity mismatch")
		}
	}
	stage = "lfs-materialisation"
	for _, args := range [][]string{{"lfs", "install", "--local", "--skip-smudge"}, {"lfs", "pull", "--include=", "--exclude=", "origin"}, {"lfs", "checkout"}, {"lfs", "fsck"}} {
		if err := run(git, args...); err != nil {
			return "", err
		}
	}
	if err := overridePR97VerifyLFS(ctx, git, project); err != nil {
		return "", err
	}
	if err := overridePR97VerifyCheckout(ctx, git, project); err != nil {
		return "", err
	}
	stage = "runtime-staging"
	staging := filepath.Join(project, "tools", "stage_rainline_runtime_for_unreal.py")
	if err := overridePR97Regular(staging); err != nil {
		return "", err
	}
	if err := run(python, staging); err != nil {
		return "", err
	}
	if err := run(python, staging, "--check"); err != nil {
		return "", err
	}
	stage = "editor-compile-and-rainline-automation"
	if err := run(pwsh, "-NoProfile", "-NonInteractive", "-File", filepath.Join(project, "tools", "windows", "validate_real_unreal.ps1"), "-UnrealRoot", engine); err != nil {
		return "", err
	}
	nativeRaw, err := overridePR97ReadEvidence(filepath.Join(project, "Saved", "Validation", "real-unreal-validation.json"))
	if err != nil {
		return "", err
	}
	native, err := decodeOverridePR97NativeReport(nativeRaw)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Clean(native.Project), projectFile) || !strings.EqualFold(filepath.Clean(native.UnrealRoot), engine) {
		return "", errors.New("PR97 native evidence names another project or engine")
	}
	stage = "win64-package-and-packaged-zeroday-runtime"
	if err := run(pwsh, "-NoProfile", "-NonInteractive", "-File", filepath.Join(project, "tools", "windows", "build_win64_review.ps1"), "-UnrealRoot", engine); err != nil {
		return "", err
	}
	packageRaw, err := overridePR97ReadEvidence(filepath.Join(project, "Saved", "Validation", "win64-review-build.json"))
	if err != nil {
		return "", err
	}
	packaged, err := decodeOverridePR97PackageReport(packageRaw)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Clean(packaged.Project), projectFile) || !strings.EqualFold(filepath.Clean(packaged.UnrealRoot), engine) {
		return "", errors.New("PR97 package evidence names another project or engine")
	}
	archive := filepath.Join(project, "Saved", "Win64Review")
	relative, err := filepath.Rel(archive, packaged.Package.Executable)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", errors.New("PR97 package executable is outside the fixed archive")
	}
	if !strings.HasPrefix(filepath.Base(packaged.Package.Executable), "OverrideZeroDay") || filepath.Ext(packaged.Package.Executable) != ".exe" {
		return "", errors.New("PR97 package executable identity mismatch")
	}
	if err := overridePR97Regular(packaged.Package.Executable); err != nil {
		return "", err
	}
	executableSHA, err := sha256RegularFile(packaged.Package.Executable)
	if err != nil {
		return "", err
	}
	if err := overridePR97VerifyCheckout(ctx, git, project); err != nil {
		return "", err
	}
	evidence["native_report_sha256"] = sha256Bytes(nativeRaw)
	evidence["package_report_sha256"] = sha256Bytes(packageRaw)
	evidence["executable_sha256"] = executableSHA
	evidence["executable_relative_path"] = filepath.ToSlash(relative)
	evidence["packaged_runtime_proved"] = true
	evidence["artifact_location"] = "Workbench/override-pr97-proof/" + id + "/project/Saved"
	stage = "completed"
	completed = true
	return "", nil
}

func overridePR97NoAlias(path string) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(path) || !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(path)) {
		return errors.New("PR97 path is unavailable or traverses a filesystem alias")
	}
	return nil
}
func overridePR97Regular(path string) error {
	if err := overridePR97NoAlias(path); err != nil {
		return err
	}
	return requireOverrideRinRegularExecutable(path)
}
func overridePR97ReadEvidence(path string) ([]byte, error) {
	if err := overridePR97Regular(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return nil, errors.New("PR97 evidence exceeds its size bound")
	}
	return bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}), nil
}
func overridePR97VerifyCheckout(ctx context.Context, git, project string) error {
	head, err := outputOverrideRinGit(ctx, git, project, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != overridePR97SourceSHA {
		return errors.New("PR97 source SHA changed")
	}
	status, err := outputOverrideRinGit(ctx, git, project, "status", "--porcelain", "--untracked-files=no")
	if err != nil || len(bytes.TrimSpace(status)) != 0 {
		return errors.New("PR97 tracked source changed")
	}
	return nil
}

// Stream the fixed LFS status command without an unbounded stdout allocation.
func overridePR97VerifyLFS(ctx context.Context, git, project string) error {
	cmd := exec.CommandContext(ctx, git, "lfs", "ls-files")
	cmd.Dir = project
	cmd.Env = overrideRinGitEnvironment()
	configureChildProcess(cmd, false)
	capture := &overridePR97LFSCapture{}
	cmd.Stdout = capture
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return errors.New("PR97 LFS status query failed")
	}
	if !capture.complete() {
		return errors.New("PR97 LFS content is missing, a pointer, or malformed")
	}
	return nil
}

// Termination targets only this operation's own process tree. No caller PID,
// command line or generic kill operation is exposed.
func overridePR97RunProcess(ctx context.Context, dir string, env []string, log io.Writer, program string, args ...string) error {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, env, log, log
	configureChildProcess(cmd, false)
	taskkill := filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe")
	if err := overridePR97Regular(taskkill); err != nil {
		return err
	}
	cmd.Cancel = func() error {
		killCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		kill := exec.CommandContext(killCtx, taskkill, "/PID", fmt.Sprint(cmd.Process.Pid), "/T", "/F")
		configureChildProcess(kill, false)
		kill.Stdout, kill.Stderr = io.Discard, io.Discard
		if err := kill.Run(); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	cmd.WaitDelay = 20 * time.Second
	return cmd.Run()
}
