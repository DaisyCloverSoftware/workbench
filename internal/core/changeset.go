package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxChangesetFiles     = 200
	maxChangesetDiffBytes = 2 << 20
	maxChangedFileBytes   = 1 << 20
)

type ChangesetInspection struct {
	Project      string   `json:"project"`
	BaseRevision string   `json:"base_revision"`
	Files        []string `json:"files"`
	Untracked    []string `json:"untracked,omitempty"`
	Diff         string   `json:"diff,omitempty"`
	Clean        bool     `json:"clean"`
	Safe         bool     `json:"safe"`
}

// InspectChangeset builds a bounded, model-safe view of local repository
// changes. It never stages, commits, pushes, or mutates the repository.
func InspectChangeset(ctx context.Context, project string) (ChangesetInspection, error) {
	root, err := projectRoot(project)
	if err != nil {
		return ChangesetInspection{}, err
	}
	gitRootRaw, err := runGitLimited(ctx, root, 16<<10, "rev-parse", "--show-toplevel")
	if err != nil {
		return ChangesetInspection{}, fmt.Errorf("project is not a readable git repository: %w", err)
	}
	gitRoot, err := filepath.EvalSymlinks(strings.TrimSpace(gitRootRaw))
	if err != nil {
		return ChangesetInspection{}, err
	}
	if filepath.Clean(gitRoot) != filepath.Clean(root) {
		return ChangesetInspection{}, errors.New("Workbench changeset inspection requires the configured project to be the git repository root")
	}
	base, err := runGitLimited(ctx, root, 16<<10, "rev-parse", "HEAD")
	if err != nil {
		return ChangesetInspection{}, fmt.Errorf("read repository baseline: %w", err)
	}
	trackedRaw, err := runGitLimited(ctx, root, maxChangesetDiffBytes, "diff", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		return ChangesetInspection{}, fmt.Errorf("list tracked changes: %w", err)
	}
	untrackedRaw, err := runGitLimited(ctx, root, maxChangesetDiffBytes, "ls-files", "--others", "--exclude-standard", "-z", "--")
	if err != nil {
		return ChangesetInspection{}, fmt.Errorf("list untracked changes: %w", err)
	}
	tracked := splitNULPaths(trackedRaw)
	untracked := splitNULPaths(untrackedRaw)
	files := append(append([]string(nil), tracked...), untracked...)
	files = uniqueSortedStrings(files)
	if len(files) > maxChangesetFiles {
		return ChangesetInspection{}, fmt.Errorf("changeset contains more than %d files", maxChangesetFiles)
	}

	inspection := ChangesetInspection{
		Project:      root,
		BaseRevision: strings.TrimSpace(base),
		Files:        files,
		Untracked:    uniqueSortedStrings(untracked),
		Clean:        len(files) == 0,
		Safe:         true,
	}
	if inspection.Clean {
		return inspection, nil
	}

	for _, rel := range files {
		if err := inspectChangedPath(root, rel); err != nil {
			return ChangesetInspection{}, err
		}
	}
	diff, err := runGitLimited(ctx, root, maxChangesetDiffBytes, "diff", "--no-ext-diff", "--no-color", "HEAD", "--")
	if err != nil {
		return ChangesetInspection{}, fmt.Errorf("read repository diff: %w", err)
	}
	if strings.Contains(diff, "Binary files ") || strings.Contains(diff, "GIT binary patch") {
		return ChangesetInspection{}, errors.New("changeset contains binary modifications; Workbench will not auto-publish them")
	}
	if LooksSecret(diff) {
		return ChangesetInspection{}, errors.New("changeset diff appears to contain secret material; Workbench will not expose or auto-publish it")
	}
	inspection.Diff = diff
	return inspection, nil
}

func inspectChangedPath(root, rel string) error {
	rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(filepath.FromSlash(rel)) {
		return errors.New("changeset contains an invalid repository path")
	}
	if sensitiveProjectPath(rel) {
		return fmt.Errorf("changeset touches a protected credential path: %s", rel)
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	st, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("changeset contains a symbolic link: %s", rel)
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("changeset path is not a regular file: %s", rel)
	}
	if st.Size() > maxChangedFileBytes {
		return fmt.Errorf("changed file exceeds %d bytes: %s", maxChangedFileBytes, rel)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return fmt.Errorf("changeset contains a binary file: %s", rel)
	}
	if LooksSecret(string(b)) {
		return fmt.Errorf("changed file appears to contain secret material: %s", rel)
	}
	return nil
}

func splitNULPaths(raw string) []string {
	parts := strings.Split(raw, "\x00")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, filepath.ToSlash(filepath.Clean(filepath.FromSlash(part))))
	}
	return out
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

type limitedCapture struct {
	buf      bytes.Buffer
	limit    int64
	exceeded bool
}

func (w *limitedCapture) Write(p []byte) (int, error) {
	n := len(p)
	remaining := w.limit - int64(w.buf.Len())
	if remaining > 0 {
		take := int64(n)
		if take > remaining {
			take = remaining
		}
		_, _ = w.buf.Write(p[:int(take)])
	}
	if int64(n) > remaining {
		w.exceeded = true
	}
	return n, nil
}

func (w *limitedCapture) String() string { return w.buf.String() }

func runGitLimited(parent context.Context, dir string, limit int64, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	configureChildProcess(cmd, false)
	stdout := &limitedCapture{limit: limit}
	stderr := &limitedCapture{limit: 64 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if stdout.exceeded {
		return "", fmt.Errorf("git output exceeded %d bytes", limit)
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return stdout.String(), nil
}
