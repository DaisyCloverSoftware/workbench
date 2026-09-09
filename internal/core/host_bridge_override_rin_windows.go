//go:build windows

package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type overrideRinProjectDocument struct {
	FileVersion       int    `json:"FileVersion"`
	EngineAssociation string `json:"EngineAssociation"`
}

func runOverrideRinCanonicalExport(ctx context.Context, detectedUnrealExecutable, jobID string) (string, error) {
	if _, err := unrealVersionFileForExecutable(detectedUnrealExecutable); err != nil {
		return "", err
	}
	jobID, err := validateHostBridgeID(jobID)
	if err != nil || !strings.HasPrefix(strings.ToLower(jobID), "hostjob_") {
		return "", errors.New("canonical Rin export job id is invalid")
	}
	gitExecutable := findOverrideRinGitExecutable()
	if gitExecutable == "" {
		return "", errors.New("Git for Windows is not installed in an allowlisted location")
	}
	if err := probeOverrideRinGitLFS(ctx, gitExecutable); err != nil {
		return "", err
	}

	workRoot, artifactRoot, cleanup, err := prepareOverrideRinRecoveryWorkspace(jobID)
	if err != nil {
		return "", err
	}
	succeeded := false
	defer func() {
		cleanup(!succeeded)
	}()
	projectRoot := filepath.Join(workRoot, "project")
	if err := os.Mkdir(projectRoot, 0o700); err != nil {
		return "", errors.New("Workbench could not create the isolated Override checkout")
	}

	setupCtx, setupCancel := context.WithTimeout(ctx, 12*time.Minute)
	defer setupCancel()
	if err := runOverrideRinGit(setupCtx, gitExecutable, "", "init", projectRoot); err != nil {
		return "", err
	}
	for _, args := range [][]string{
		{"remote", "add", "origin", overrideRinRepositoryURL},
		{"config", "core.autocrlf", "false"},
		{"config", "advice.detachedHead", "false"},
	} {
		if err := runOverrideRinGit(setupCtx, gitExecutable, projectRoot, args...); err != nil {
			return "", err
		}
	}
	if err := runOverrideRinGit(setupCtx, gitExecutable, projectRoot, "fetch", "--no-tags", "--depth=2", "origin", overrideRinExporterSHA); err != nil {
		return "", err
	}
	exporter, err := outputOverrideRinGit(setupCtx, gitExecutable, projectRoot, "show", overrideRinExporterSHA+":"+overrideRinExporterPath)
	if err != nil || len(exporter) == 0 || len(exporter) > 128<<10 {
		return "", errors.New("the pinned canonical Rin exporter could not be materialised")
	}
	exporterPath := filepath.Join(workRoot, "export_canonical_rin_inputs.py")
	if err := os.WriteFile(exporterPath, exporter, 0o600); err != nil {
		return "", errors.New("Workbench could not stage the pinned canonical Rin exporter")
	}
	exporterSHA, err := sha256RegularFile(exporterPath)
	if err != nil {
		return "", err
	}
	if err := runOverrideRinGit(setupCtx, gitExecutable, projectRoot, "fetch", "--no-tags", "--depth=1", "origin", overrideRinSourceSHA); err != nil {
		return "", err
	}
	if err := runOverrideRinGit(setupCtx, gitExecutable, projectRoot, "checkout", "--detach", "--force", overrideRinSourceSHA); err != nil {
		return "", err
	}
	if err := runOverrideRinGit(setupCtx, gitExecutable, projectRoot, "lfs", "install", "--local", "--skip-smudge"); err != nil {
		return "", err
	}
	// Only MetaHuman content is materialised. The exporter independently verifies
	// every canonical Rin identity package against its committed LFS pin.
	if err := runOverrideRinGit(setupCtx, gitExecutable, projectRoot, "lfs", "pull", "--include=Content/MetaHumans/**", "origin"); err != nil {
		return "", err
	}
	if err := verifyOverrideRinCheckout(setupCtx, gitExecutable, projectRoot); err != nil {
		return "", err
	}

	projectPath := filepath.Join(projectRoot, overrideRinProjectName)
	if err := validateOverrideRinProject(projectPath); err != nil {
		return "", err
	}
	engineRoot, err := resolveOverrideRinAssociatedEngine(ctx, overrideRinEngineAssociation)
	if err != nil {
		return "", err
	}
	editor := filepath.Join(engineRoot, "Engine", "Binaries", "Win64", "UnrealEditor-Cmd.exe")
	ubt := filepath.Join(engineRoot, "Engine", "Binaries", "DotNET", "UnrealBuildTool", "UnrealBuildTool.exe")
	if err := requireOverrideRinRegularExecutable(editor); err != nil {
		return "", errors.New("the project-associated UnrealEditor-Cmd executable is unavailable")
	}
	if err := requireOverrideRinRegularExecutable(ubt); err != nil {
		return "", errors.New("the project-associated UnrealBuildTool executable is unavailable")
	}
	engineVersion, err := runUnrealVersion(editor)
	if err != nil {
		return "", err
	}

	buildCtx, buildCancel := context.WithTimeout(ctx, 28*time.Minute)
	buildErr := runOverrideRinProcess(buildCtx, projectRoot, nil, ubt,
		overrideRinEditorTarget, "Win64", "Development",
		"-Project="+projectPath, "-WaitMutex", "-NoHotReloadFromIDE", "-NoEngineChanges")
	buildCancel()
	if buildErr != nil {
		return "", errors.New("the exact-head Override editor build failed")
	}
	if err := verifyOverrideRinCheckout(ctx, gitExecutable, projectRoot); err != nil {
		return "", err
	}

	exportCtx, exportCancel := context.WithTimeout(ctx, 15*time.Minute)
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=Never",
		"PATH="+filepath.Dir(gitExecutable)+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	exportErr := runOverrideRinProcess(exportCtx, projectRoot, env, editor,
		projectPath,
		"-ExecutePythonScript="+exporterPath,
		"-unattended", "-stdout", "-nop4", "-nosplash", "-NoEpicPortal", "-nocrashreports", "-nullrhi")
	exportCancel()
	if exportErr != nil {
		return "", errors.New("the canonical Rin Unreal export process failed")
	}

	manifestPath, fbxPath, err := findOverrideRinExportEvidence(projectRoot)
	if err != nil {
		return "", err
	}
	rawManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", errors.New("canonical Rin export manifest could not be read")
	}
	manifest, err := validateOverrideRinExportManifest(rawManifest, fbxPath, exporterSHA)
	if err != nil {
		return "", err
	}
	if err := verifyOverrideRinCheckout(ctx, gitExecutable, projectRoot); err != nil {
		return "", err
	}

	artifactManifest := filepath.Join(artifactRoot, overrideRinManifestName)
	artifactFBX := filepath.Join(artifactRoot, overrideRinFBXName)
	if err := copyOverrideRinRegularFile(manifestPath, artifactManifest); err != nil {
		return "", err
	}
	if err := copyOverrideRinRegularFile(fbxPath, artifactFBX); err != nil {
		return "", err
	}
	manifestSHA, err := sha256RegularFile(artifactManifest)
	if err != nil {
		return "", err
	}
	fbxSHA, err := sha256RegularFile(artifactFBX)
	if err != nil || fbxSHA != manifest.FBX.SHA256 {
		return "", errors.New("preserved canonical Rin FBX failed its final hash check")
	}
	result := overrideRinExportResult{
		SchemaVersion: 1, ArtifactID: jobID, SourceSHA: manifest.SourceSHA,
		EngineVersion: engineVersion, Body: manifest.Body, Face: manifest.Face,
		Groom: manifest.Groom, Blueprint: manifest.Blueprint, Skeleton: manifest.Skeleton,
		IdentityAssetCount: len(manifest.Assets), MaterialSlotCount: len(manifest.Materials),
		FBXBytes: manifest.FBX.Bytes, FBXSHA256: fbxSHA, ManifestSHA256: manifestSHA,
		InputExportCompleted: true, VisualAcceptance: "not_assessed", AnimationAcceptance: "not_assessed",
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", errors.New("canonical Rin export result could not be encoded")
	}
	succeeded = true
	return string(encoded), nil
}

func findOverrideRinGitExecutable() string {
	seen := map[string]bool{}
	var candidates []string
	for _, root := range []string{os.Getenv("ProgramW6432"), os.Getenv("ProgramFiles")} {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(root, "Git", "cmd", "git.exe"),
			filepath.Join(root, "Git", "bin", "git.exe"))
	}
	if found, err := exec.LookPath("git.exe"); err == nil {
		candidates = append(candidates, found)
	}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		key := strings.ToLower(candidate)
		if seen[key] {
			continue
		}
		seen[key] = true
		if requireOverrideRinRegularExecutable(candidate) == nil {
			return candidate
		}
	}
	return ""
}

func probeOverrideRinGitLFS(ctx context.Context, gitExecutable string) error {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := runOverrideRinProcess(probeCtx, "", overrideRinGitEnvironment(), gitExecutable, "lfs", "version"); err != nil {
		return errors.New("Git LFS is unavailable for the isolated canonical Rin checkout")
	}
	return nil
}

func overrideRinGitEnvironment() []string {
	env := append([]string{}, os.Environ()...)
	return append(env, "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never", "GIT_LFS_SKIP_SMUDGE=1")
}

func runOverrideRinGit(ctx context.Context, gitExecutable, dir string, args ...string) error {
	return runOverrideRinProcess(ctx, dir, overrideRinGitEnvironment(), gitExecutable, args...)
}

func outputOverrideRinGit(ctx context.Context, gitExecutable, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, gitExecutable, args...)
	cmd.Dir = dir
	cmd.Env = overrideRinGitEnvironment()
	configureChildProcess(cmd, false)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, errors.New("required pinned Git materialisation failed")
	}
	if stdout.Len() > 128<<10 {
		return nil, errors.New("pinned Git materialisation exceeded its output bound")
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func runOverrideRinProcess(ctx context.Context, dir string, env []string, program string, args ...string) error {
	cmd := exec.CommandContext(ctx, program, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	configureChildProcess(cmd, false)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return errors.New("bounded recovery process timed out or was cancelled")
		}
		return errors.New("bounded recovery process failed")
	}
	return nil
}

func prepareOverrideRinRecoveryWorkspace(jobID string) (string, string, func(bool), error) {
	cache, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(cache) == "" {
		return "", "", nil, errors.New("Workbench could not resolve its recovery cache")
	}
	base := filepath.Join(cache, "Workbench", "override-rin-recovery")
	for _, dir := range []string{base, filepath.Join(base, "work"), filepath.Join(base, "artifacts")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", "", nil, errors.New("Workbench could not prepare its recovery cache")
		}
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil || !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(dir)) {
			return "", "", nil, errors.New("Workbench recovery cache resolves through an alias")
		}
	}
	work := filepath.Join(base, "work", jobID)
	artifact := filepath.Join(base, "artifacts", jobID)
	if err := os.Mkdir(work, 0o700); err != nil {
		return "", "", nil, errors.New("canonical Rin recovery workspace already exists or could not be created")
	}
	if err := os.Mkdir(artifact, 0o700); err != nil {
		_ = os.RemoveAll(work)
		return "", "", nil, errors.New("canonical Rin artifact workspace already exists or could not be created")
	}
	cleanup := func(removeArtifact bool) {
		_ = os.RemoveAll(work)
		if removeArtifact {
			_ = os.RemoveAll(artifact)
		}
	}
	return work, artifact, cleanup, nil
}

func validateOverrideRinProject(projectPath string) error {
	info, err := os.Lstat(projectPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 256<<10 {
		return errors.New("exact-head Override project file is missing or invalid")
	}
	raw, err := os.ReadFile(projectPath)
	if err != nil {
		return errors.New("exact-head Override project file could not be read")
	}
	var document overrideRinProjectDocument
	if err := json.Unmarshal(raw, &document); err != nil || document.FileVersion != 3 || document.EngineAssociation != overrideRinEngineAssociation {
		return errors.New("exact-head Override project uses an unexpected engine association")
	}
	return nil
}

func resolveOverrideRinAssociatedEngine(ctx context.Context, association string) (string, error) {
	systemRoot := strings.TrimSpace(os.Getenv("SystemRoot"))
	if systemRoot == "" {
		return "", errors.New("Windows system root is unavailable")
	}
	reg := filepath.Join(systemRoot, "System32", "reg.exe")
	if requireOverrideRinRegularExecutable(reg) != nil {
		return "", errors.New("Windows registry query utility is unavailable")
	}
	queryCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(queryCtx, reg, "QUERY", `HKCU\SOFTWARE\Epic Games\Unreal Engine\Builds`, "/v", association)
	configureChildProcess(cmd, false)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 || len(out) > 64<<10 {
		return "", errors.New("the exact project engine association is not registered for this Windows user")
	}
	for _, line := range strings.Split(string(out), "\n") {
		upper := strings.ToUpper(line)
		marker := "REG_SZ"
		idx := strings.Index(upper, marker)
		if idx < 0 {
			continue
		}
		root := filepath.Clean(strings.TrimSpace(line[idx+len(marker):]))
		if root == "." || !filepath.IsAbs(root) || strings.ContainsAny(root, "\r\n\x00") {
			continue
		}
		return root, nil
	}
	return "", errors.New("the exact project engine association is malformed")
}

func requireOverrideRinRegularExecutable(path string) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("required executable is missing or invalid")
	}
	return nil
}

func verifyOverrideRinCheckout(ctx context.Context, gitExecutable, projectRoot string) error {
	head, err := outputOverrideRinGit(ctx, gitExecutable, projectRoot, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != overrideRinSourceSHA {
		return errors.New("isolated Override checkout is not at the exact recovery source SHA")
	}
	status, err := outputOverrideRinGit(ctx, gitExecutable, projectRoot, "status", "--porcelain", "--untracked-files=all")
	if err != nil || strings.TrimSpace(string(status)) != "" {
		return errors.New("isolated Override checkout is not clean")
	}
	return nil
}

func findOverrideRinExportEvidence(projectRoot string) (string, string, error) {
	pattern := filepath.Join(projectRoot, "Saved", "Recovery", "RinInputs", "*", overrideRinManifestName)
	manifests, err := filepath.Glob(pattern)
	if err != nil || len(manifests) != 1 {
		return "", "", errors.New("canonical Rin exporter did not produce exactly one evidence manifest")
	}
	manifest := manifests[0]
	fbx := filepath.Join(filepath.Dir(manifest), overrideRinFBXName)
	return manifest, fbx, nil
}

func copyOverrideRinRegularFile(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("canonical Rin recovery artifact is missing or invalid")
	}
	in, err := os.Open(source)
	if err != nil {
		return errors.New("canonical Rin recovery artifact could not be opened")
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("canonical Rin recovery artifact could not be preserved")
	}
	copied, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || copied != info.Size() {
		_ = os.Remove(destination)
		return errors.New("canonical Rin recovery artifact copy was incomplete")
	}
	return nil
}

func sha256Bytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
