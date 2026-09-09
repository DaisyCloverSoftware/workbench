//go:build windows

package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	overrideRinWardrobeMaxFirstLevelDirs = 256
	overrideRinWardrobeMaxSecondLevelDirs = 1024
	overrideRinWardrobeMaxWalkEntries = 250000
	overrideRinWardrobeMaxRetainedMatches = 1024
	overrideRinWardrobeMaxReturnedCandidates = 32
	overrideRinWardrobeMaxHashBytesPerFile int64 = 512 << 20
	overrideRinWardrobeMaxHashBytesTotal int64 = 2 << 30
)

type overrideRinWardrobeDiscoveredCandidate struct {
	absolute  string
	relative  string
	extension string
	bytes     int64
	score     int
}

func runOverrideRinWardrobeInventory(ctx context.Context, jobID string) (string, error) {
	jobID, err := validateHostBridgeID(jobID)
	if err != nil || !strings.HasPrefix(strings.ToLower(jobID), "hostjob_") {
		return "", errors.New("Rin wardrobe inventory job id is invalid")
	}
	gitExecutable := findOverrideRinGitExecutable()
	if gitExecutable == "" {
		return "", errors.New("Git for Windows is not installed in an allowlisted location")
	}
	result, err := inspectOverrideRinWardrobeRoot(ctx, gitExecutable, overrideRinWardrobeReposRoot, jobID)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > 15<<10 {
		return "", errors.New("Rin wardrobe inventory result exceeded its privacy-safe output bound")
	}
	return string(encoded), nil
}

func inspectOverrideRinWardrobeRoot(ctx context.Context, gitExecutable, reposRoot, jobID string) (overrideRinWardrobeInventoryResult, error) {
	root, err := validateOverrideRinWardrobeReposRoot(reposRoot)
	if err != nil {
		return overrideRinWardrobeInventoryResult{}, err
	}
	worktree, head, err := findOverrideRinHardlineWorktree(ctx, gitExecutable, root)
	if err != nil {
		return overrideRinWardrobeInventoryResult{}, err
	}
	matches, matchedCount, truncated, err := scanOverrideRinWardrobeCandidates(ctx, worktree)
	if err != nil {
		return overrideRinWardrobeInventoryResult{}, err
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return strings.ToLower(matches[i].relative) < strings.ToLower(matches[j].relative)
	})
	if len(matches) > overrideRinWardrobeMaxReturnedCandidates {
		matches = matches[:overrideRinWardrobeMaxReturnedCandidates]
		truncated = true
	}

	candidates := make([]overrideRinWardrobeCandidate, 0, len(matches))
	var hashedBytes int64
	oversized := 0
	for _, match := range matches {
		if err := ctx.Err(); err != nil {
			return overrideRinWardrobeInventoryResult{}, errors.New("Rin wardrobe inventory was cancelled")
		}
		candidate := overrideRinWardrobeCandidate{
			Path: filepath.ToSlash(match.relative), Extension: match.extension,
			Bytes: match.bytes, HashStatus: "not_hashed", Score: match.score,
		}
		candidate.Tracked = overrideRinWardrobePathTracked(ctx, gitExecutable, worktree, match.relative)
		if match.bytes <= overrideRinWardrobeMaxHashBytesPerFile && hashedBytes+match.bytes <= overrideRinWardrobeMaxHashBytesTotal {
			digest, hashErr := sha256RegularFile(match.absolute)
			if hashErr != nil {
				return overrideRinWardrobeInventoryResult{}, errors.New("a selected Rin wardrobe candidate could not be hashed as a regular file")
			}
			candidate.SHA256 = digest
			candidate.HashStatus = "complete"
			hashedBytes += match.bytes
		} else {
			candidate.HashStatus = "omitted_size_bound"
			oversized++
		}
		candidates = append(candidates, candidate)
	}

	return overrideRinWardrobeInventoryResult{
		SchemaVersion: 1, ArtifactID: jobID,
		ReposRoot: root, WorktreeRoot: worktree, Branch: overrideRinWardrobeBranch, HeadSHA: head,
		ReadOnly: true, MutationPerformed: false,
		MatchedCount: matchedCount, ReturnedCount: len(candidates), OversizedCount: oversized,
		CandidateTruncated: truncated, Candidates: candidates,
		VisualAcceptance: "not_assessed", AnimationAcceptance: "not_assessed",
	}, nil
}

func validateOverrideRinWardrobeReposRoot(root string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || !filepath.IsAbs(root) {
		return "", errors.New("Rin wardrobe repository root is invalid")
	}
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("Rin wardrobe repository root is unavailable or aliased")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("Rin wardrobe repository root could not be canonicalised")
	}
	resolved = filepath.Clean(resolved)
	resolvedInfo, err := os.Lstat(resolved)
	if err != nil || resolvedInfo.Mode()&os.ModeSymlink != 0 || !resolvedInfo.IsDir() {
		return "", errors.New("Rin wardrobe canonical repository root is unavailable or aliased")
	}
	return resolved, nil
}

func findOverrideRinHardlineWorktree(ctx context.Context, gitExecutable, reposRoot string) (string, string, error) {
	dirs, err := boundedOverrideRinRepositoryDirs(reposRoot)
	if err != nil {
		return "", "", err
	}
	var found string
	var head string
	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return "", "", errors.New("Rin wardrobe worktree discovery was cancelled")
		}
		if !overrideRinHasGitMarker(dir) {
			continue
		}
		branch, err := outputOverrideRinWardrobeGit(ctx, gitExecutable, dir, "symbolic-ref", "--quiet", "--short", "HEAD")
		if err != nil || strings.TrimSpace(branch) != overrideRinWardrobeBranch {
			continue
		}
		origin, err := outputOverrideRinWardrobeGit(ctx, gitExecutable, dir, "remote", "get-url", "origin")
		if err != nil || !overrideRinWardrobeOriginAllowed(strings.TrimSpace(origin)) {
			continue
		}
		candidateHead, err := outputOverrideRinWardrobeGit(ctx, gitExecutable, dir, "rev-parse", "HEAD")
		candidateHead = strings.TrimSpace(candidateHead)
		if err != nil || !overrideRinWardrobeValidSHA(candidateHead) {
			return "", "", errors.New("the protected Hardline worktree has no valid HEAD")
		}
		if found != "" {
			return "", "", errors.New("multiple protected Hardline worktrees matched the sealed inventory contract")
		}
		found = dir
		head = strings.ToLower(candidateHead)
	}
	if found == "" {
		return "", "", errors.New("the protected Hardline Override worktree was not found beneath the sealed repository root")
	}
	return found, head, nil
}

func boundedOverrideRinRepositoryDirs(root string) ([]string, error) {
	var dirs []string
	first, err := os.ReadDir(root)
	if err != nil {
		return nil, errors.New("Rin wardrobe repository root could not be enumerated")
	}
	if len(first) > overrideRinWardrobeMaxFirstLevelDirs {
		return nil, errors.New("Rin wardrobe repository root exceeds the bounded directory count")
	}
	for _, entry := range first {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		child := filepath.Join(root, entry.Name())
		dirs = append(dirs, child)
		second, readErr := os.ReadDir(child)
		if readErr != nil {
			continue
		}
		for _, nested := range second {
			if len(dirs) >= overrideRinWardrobeMaxSecondLevelDirs {
				return nil, errors.New("Rin wardrobe repository discovery exceeded its bounded directory count")
			}
			if !nested.IsDir() || nested.Type()&os.ModeSymlink != 0 {
				continue
			}
			dirs = append(dirs, filepath.Join(child, nested.Name()))
		}
	}
	return dirs, nil
}

func overrideRinHasGitMarker(dir string) bool {
	info, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil && info.Mode()&os.ModeSymlink == 0 && (info.IsDir() || info.Mode().IsRegular())
}

func overrideRinWardrobeOriginAllowed(raw string) bool {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "git@github.com:") {
		path := raw[len("git@github.com:"):]
		return overrideRinWardrobeGitHubPathAllowed(path)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || !strings.EqualFold(u.Hostname(), "github.com") || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		if u.User != nil {
			return false
		}
	case "ssh":
		if u.User == nil || u.User.Username() != "git" {
			return false
		}
	default:
		return false
	}
	return overrideRinWardrobeGitHubPathAllowed(strings.TrimPrefix(u.Path, "/"))
}

func overrideRinWardrobeGitHubPathAllowed(path string) bool {
	path = strings.TrimSuffix(strings.TrimSpace(path), ".git")
	return strings.EqualFold(path, "DaisyCloverSoftware/override")
}

func overrideRinWardrobeValidSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func scanOverrideRinWardrobeCandidates(ctx context.Context, worktree string) ([]overrideRinWardrobeDiscoveredCandidate, int, bool, error) {
	matches := make([]overrideRinWardrobeDiscoveredCandidate, 0, 64)
	matchedCount := 0
	walkEntries := 0
	truncated := false
	err := filepath.WalkDir(worktree, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		walkEntries++
		if walkEntries > overrideRinWardrobeMaxWalkEntries {
			return errors.New("Rin wardrobe worktree exceeds the bounded scan size")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if path != worktree && overrideRinWardrobeSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(worktree, path)
		if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			return nil
		}
		score := overrideRinWardrobeCandidateScore(relative)
		if score == 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 0 {
			return nil
		}
		matchedCount++
		if len(matches) >= overrideRinWardrobeMaxRetainedMatches {
			truncated = true
			return nil
		}
		matches = append(matches, overrideRinWardrobeDiscoveredCandidate{
			absolute: path, relative: relative, extension: strings.ToLower(filepath.Ext(relative)),
			bytes: info.Size(), score: score,
		})
		return nil
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, false, errors.New("Rin wardrobe candidate scan was cancelled")
		}
		return nil, 0, false, err
	}
	return matches, matchedCount, truncated, nil
}

func overrideRinWardrobeSkipDir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".git", ".vs", "binaries", "deriveddatacache", "intermediate", "saved", "node_modules":
		return true
	default:
		return false
	}
}

func overrideRinWardrobePathTracked(ctx context.Context, gitExecutable, worktree, relative string) bool {
	cmd := exec.CommandContext(ctx, gitExecutable, "ls-files", "--error-unmatch", "--", relative)
	cmd.Dir = worktree
	cmd.Env = overrideRinWardrobeGitEnvironment()
	configureChildProcess(cmd, false)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func outputOverrideRinWardrobeGit(ctx context.Context, gitExecutable, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, gitExecutable, args...)
	cmd.Dir = dir
	cmd.Env = overrideRinWardrobeGitEnvironment()
	configureChildProcess(cmd, false)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", errors.New("bounded read-only Git inspection failed")
	}
	if stdout.Len() > 64<<10 {
		return "", errors.New("bounded read-only Git inspection exceeded its output limit")
	}
	return stdout.String(), nil
}

func overrideRinWardrobeGitEnvironment() []string {
	env := append([]string{}, os.Environ()...)
	return append(env, "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never", "GIT_OPTIONAL_LOCKS=0", "GIT_LFS_SKIP_SMUDGE=1")
}
