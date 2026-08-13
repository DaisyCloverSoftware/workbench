package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"
)

type PreparedChangeset struct {
	Project      string   `json:"project"`
	BaseRevision string   `json:"base_revision"`
	Fingerprint string   `json:"fingerprint"`
	Branch       string   `json:"branch"`
	Commit       string   `json:"commit"`
	Files        []string `json:"files"`
}

// PrepareChangeset freezes the current safe changeset into an isolated
// Workbench-owned local commit and branch. It never stages the user's active
// index, switches the user's branch, pushes a remote, or deploys anything.
func PrepareChangeset(ctx context.Context, project, taskID string) (PreparedChangeset, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return PreparedChangeset{}, errors.New("task id is required to prepare a changeset")
	}

	snapshot, err := SnapshotChangeset(ctx, project)
	if err != nil {
		return PreparedChangeset{}, err
	}
	if snapshot.Inspection.Clean {
		return PreparedChangeset{}, errors.New("changeset is clean; there is nothing to prepare")
	}
	root := snapshot.Inspection.Project
	branch := preparedBranchName(taskID, snapshot.Fingerprint)

	parent, err := os.MkdirTemp("", "workbench-prepare-")
	if err != nil {
		return PreparedChangeset{}, err
	}
	defer os.RemoveAll(parent)
	worktree := filepath.Join(parent, "tree")
	if err := runGitMutation(ctx, root, "worktree", "add", "--detach", "--quiet", worktree, snapshot.Inspection.BaseRevision); err != nil {
		return PreparedChangeset{}, fmt.Errorf("create isolated preparation worktree: %w", err)
	}
	defer func() {
		_ = exec.Command("git", "-C", root, "worktree", "remove", "--force", worktree).Run()
	}()

	if err := reproduceSnapshotFiles(root, worktree, snapshot.Inspection.Files); err != nil {
		return PreparedChangeset{}, err
	}
	preparedSnapshot, err := SnapshotChangeset(ctx, worktree)
	if err != nil {
		return PreparedChangeset{}, fmt.Errorf("verify isolated changeset: %w", err)
	}
	if preparedSnapshot.Fingerprint != snapshot.Fingerprint {
		return PreparedChangeset{}, errors.New("changeset changed while Workbench was preparing it")
	}

	addArgs := append([]string{"add", "-A", "--"}, snapshot.Inspection.Files...)
	if err := runGitMutation(ctx, worktree, addArgs...); err != nil {
		return PreparedChangeset{}, fmt.Errorf("stage isolated changeset: %w", err)
	}
	stagedRaw, err := runGitLimited(ctx, worktree, maxChangesetDiffBytes, "diff", "--cached", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		return PreparedChangeset{}, fmt.Errorf("verify staged file set: %w", err)
	}
	staged := uniqueSortedStrings(splitNULPaths(stagedRaw))
	if !reflect.DeepEqual(staged, snapshot.Inspection.Files) {
		return PreparedChangeset{}, errors.New("staged file set differs from the inspected changeset")
	}
	cachedDiff, err := runGitLimited(ctx, worktree, maxChangesetDiffBytes, "diff", "--cached", "--no-ext-diff", "--no-color", "HEAD", "--")
	if err != nil {
		return PreparedChangeset{}, fmt.Errorf("verify staged diff: %w", err)
	}
	if strings.Contains(cachedDiff, "Binary files ") || strings.Contains(cachedDiff, "GIT binary patch") || LooksSecret(cachedDiff) {
		return PreparedChangeset{}, errors.New("staged changeset failed Workbench's publication safety check")
	}

	if err := runGitMutation(ctx, worktree,
		"-c", "user.name=Workbench Publisher",
		"-c", "user.email=workbench-publisher@users.noreply.github.com",
		"commit", "--quiet", "-m", "Workbench prepared changeset"); err != nil {
		return PreparedChangeset{}, fmt.Errorf("create isolated changeset commit: %w", err)
	}
	commit, err := runGitLimited(ctx, worktree, 16<<10, "rev-parse", "HEAD")
	if err != nil {
		return PreparedChangeset{}, err
	}
	commit = strings.TrimSpace(commit)

	ref := "refs/heads/" + branch
	if existing, ok, err := localRefSHA(ctx, root, ref); err != nil {
		return PreparedChangeset{}, err
	} else if ok {
		existingTree, treeErr := runGitLimited(ctx, root, 16<<10, "rev-parse", existing+"^{tree}")
		newTree, newTreeErr := runGitLimited(ctx, root, 16<<10, "rev-parse", commit+"^{tree}")
		if treeErr != nil || newTreeErr != nil || strings.TrimSpace(existingTree) != strings.TrimSpace(newTree) {
			return PreparedChangeset{}, fmt.Errorf("Workbench branch already exists with different content: %s", branch)
		}
		commit = existing
	} else if err := runGitMutation(ctx, root, "branch", branch, commit); err != nil {
		return PreparedChangeset{}, fmt.Errorf("create Workbench local branch: %w", err)
	}

	return PreparedChangeset{
		Project:      root,
		BaseRevision: snapshot.Inspection.BaseRevision,
		Fingerprint: snapshot.Fingerprint,
		Branch:       branch,
		Commit:       commit,
		Files:        append([]string(nil), snapshot.Inspection.Files...),
	}, nil
}

func reproduceSnapshotFiles(sourceRoot, destRoot string, files []string) error {
	for _, rel := range files {
		source := filepath.Join(sourceRoot, filepath.FromSlash(rel))
		dest := filepath.Join(destRoot, filepath.FromSlash(rel))
		st, err := os.Lstat(source)
		if errors.Is(err, os.ErrNotExist) {
			if removeErr := os.Remove(dest); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return removeErr
			}
			continue
		}
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return fmt.Errorf("changed path is no longer a regular file: %s", rel)
		}
		b, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, b, st.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func preparedBranchName(taskID, fingerprint string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(taskID) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
		if b.Len() >= 40 {
			break
		}
	}
	name := strings.Trim(b.String(), "-._")
	if name == "" {
		name = "change"
	}
	short := fingerprint
	if len(short) > 12 {
		short = short[:12]
	}
	return "workbench/" + name + "-" + short
}

func runGitMutation(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	configureChildProcess(cmd, false)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(out.String())
		if message == "" {
			message = err.Error()
		}
		return errors.New(message)
	}
	return nil
}

func localRefSHA(ctx context.Context, root, ref string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "show-ref", "--verify", "--hash", ref)
	configureChildProcess(cmd, false)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err == nil {
		return strings.TrimSpace(out.String()), true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return "", false, nil
	}
	return "", false, err
}
