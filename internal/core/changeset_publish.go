package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type PublishedChangeset struct {
	Branch         string `json:"branch"`
	Commit         string `json:"commit"`
	AlreadyPresent bool   `json:"already_present,omitempty"`
}

var scpStyleGitRemote = regexp.MustCompile(`^(?:[A-Za-z0-9._-]+@)?[A-Za-z0-9.-]+:[A-Za-z0-9._~/-]+$`)

// PublishPreparedChangeset pushes exactly one already-prepared Workbench-owned
// commit to its Workbench-owned branch. The remote target is an explicit local
// policy input; it is never read from a worker-controlled task or repository
// remote configuration. The user's current branch is never pushed and force
// updates are not permitted.
func PublishPreparedChangeset(ctx context.Context, prepared PreparedChangeset, remoteURL string) (PublishedChangeset, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if err := validatePublishRemote(remoteURL); err != nil {
		return PublishedChangeset{}, err
	}
	root, err := verifyPreparedChangeset(ctx, prepared)
	if err != nil {
		return PublishedChangeset{}, err
	}

	gitDir, err := isolatedPublishGitDir(ctx, root)
	if err != nil {
		return PublishedChangeset{}, err
	}
	defer os.RemoveAll(gitDir)
	ref := "refs/heads/" + prepared.Branch

	remoteHead, exists, err := remoteBranchSHA(ctx, gitDir, remoteURL, ref)
	if err != nil {
		return PublishedChangeset{}, err
	}
	if exists {
		if remoteHead != prepared.Commit {
			return PublishedChangeset{}, fmt.Errorf("remote Workbench branch already exists with different content: %s", prepared.Branch)
		}
		return PublishedChangeset{Branch: prepared.Branch, Commit: prepared.Commit, AlreadyPresent: true}, nil
	}

	cmd := publishGitCommand(ctx, gitDir, "push", "--no-verify", "--porcelain", remoteURL, prepared.Commit+":"+ref)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(out.String())
		if message == "" {
			message = err.Error()
		}
		return PublishedChangeset{}, fmt.Errorf("publish Workbench branch: %s", message)
	}
	return PublishedChangeset{Branch: prepared.Branch, Commit: prepared.Commit}, nil
}

func verifyPreparedChangeset(ctx context.Context, prepared PreparedChangeset) (string, error) {
	root, err := projectRoot(prepared.Project)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(prepared.Branch) == "" || !strings.HasPrefix(prepared.Branch, "workbench/") {
		return "", errors.New("prepared branch must be Workbench-owned")
	}
	if err := runGitMutation(ctx, root, "check-ref-format", "refs/heads/"+prepared.Branch); err != nil {
		return "", errors.New("prepared branch is not a valid Git ref")
	}
	if strings.TrimSpace(prepared.Commit) == "" || strings.TrimSpace(prepared.BaseRevision) == "" || strings.TrimSpace(prepared.Fingerprint) == "" {
		return "", errors.New("prepared changeset is incomplete")
	}
	if refSHA, ok, err := localRefSHA(ctx, root, "refs/heads/"+prepared.Branch); err != nil {
		return "", err
	} else if !ok || refSHA != prepared.Commit {
		return "", errors.New("prepared Workbench branch no longer points at the recorded commit")
	}

	identity, err := runGitLimited(ctx, root, 64<<10, "show", "-s", "--format=%an%x00%ae%x00%cn%x00%ce%x00%B", prepared.Commit)
	if err != nil {
		return "", fmt.Errorf("read prepared commit identity: %w", err)
	}
	parts := strings.SplitN(identity, "\x00", 5)
	if len(parts) != 5 || parts[0] != "Workbench Publisher" || parts[1] != "workbench-publisher@users.noreply.github.com" || parts[2] != "Workbench Publisher" || parts[3] != "workbench-publisher@users.noreply.github.com" || strings.TrimSpace(parts[4]) != "Workbench prepared changeset" {
		return "", errors.New("prepared commit was not created by Workbench's deterministic publisher")
	}

	parents, err := runGitLimited(ctx, root, 64<<10, "rev-list", "--parents", "-n", "1", prepared.Commit)
	if err != nil {
		return "", err
	}
	parentFields := strings.Fields(parents)
	if len(parentFields) != 2 || parentFields[0] != prepared.Commit || parentFields[1] != prepared.BaseRevision {
		return "", errors.New("prepared commit baseline does not match its recorded provenance")
	}

	changedRaw, err := runGitLimited(ctx, root, maxChangesetDiffBytes, "diff", "--name-only", "-z", prepared.BaseRevision, prepared.Commit, "--")
	if err != nil {
		return "", err
	}
	changed := uniqueSortedStrings(splitNULPaths(changedRaw))
	expected := uniqueSortedStrings(append([]string(nil), prepared.Files...))
	if !reflect.DeepEqual(changed, expected) {
		return "", errors.New("prepared commit file set no longer matches its recorded changeset")
	}
	if len(changed) == 0 || len(changed) > maxChangesetFiles {
		return "", errors.New("prepared commit has an invalid changed-file count")
	}

	diff, err := runGitLimited(ctx, root, maxChangesetDiffBytes, "diff", "--no-ext-diff", "--no-color", prepared.BaseRevision, prepared.Commit, "--")
	if err != nil {
		return "", err
	}
	if strings.Contains(diff, "Binary files ") || strings.Contains(diff, "GIT binary patch") || LooksSecret(diff) {
		return "", errors.New("prepared commit failed Workbench's final diff safety check")
	}
	for _, rel := range changed {
		if sensitiveProjectPath(rel) {
			return "", fmt.Errorf("prepared commit contains protected path: %s", rel)
		}
		if err := inspectPreparedBlob(ctx, root, prepared.Commit, rel); err != nil {
			return "", err
		}
	}
	return root, nil
}

func inspectPreparedBlob(ctx context.Context, root, commit, rel string) error {
	entry, err := runGitLimited(ctx, root, 64<<10, "ls-tree", "-z", commit, "--", rel)
	if err != nil {
		return err
	}
	if entry == "" {
		// Deleted paths have no blob in the prepared tree. Their removed content
		// has already passed the bounded diff safety check above.
		return nil
	}
	meta := strings.SplitN(entry, "\t", 2)
	if len(meta) != 2 {
		return fmt.Errorf("cannot parse prepared tree entry for %s", rel)
	}
	fields := strings.Fields(meta[0])
	if len(fields) < 3 || (fields[0] != "100644" && fields[0] != "100755") || fields[1] != "blob" {
		return fmt.Errorf("prepared path is not a regular file: %s", rel)
	}
	sizeText, err := runGitLimited(ctx, root, 64<<10, "cat-file", "-s", commit+":"+rel)
	if err != nil {
		return err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(sizeText), 10, 64)
	if err != nil || size < 0 || size > maxChangesetFileBytes {
		return fmt.Errorf("prepared file is too large or invalid: %s", rel)
	}
	content, err := runGitLimited(ctx, root, maxChangesetFileBytes+1, "show", commit+":"+rel)
	if err != nil {
		return err
	}
	if strings.IndexByte(content, 0) >= 0 {
		return fmt.Errorf("prepared file is binary: %s", rel)
	}
	if LooksSecret(content) {
		return fmt.Errorf("prepared file contains probable secret material: %s", rel)
	}
	return nil
}

func validatePublishRemote(remote string) error {
	if remote == "" {
		return errors.New("publication target is empty")
	}
	if strings.ContainsAny(remote, "\r\n\x00") || LooksSecret(remote) {
		return errors.New("publication target is unsafe or appears to contain credentials")
	}
	if filepath.IsAbs(remote) {
		return nil
	}
	if scpStyleGitRemote.MatchString(remote) {
		return nil
	}
	u, err := url.Parse(remote)
	if err != nil {
		return errors.New("publication target is not a valid URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		if u.Host == "" || u.User != nil {
			return errors.New("HTTPS publication target must not contain userinfo or embedded credentials")
		}
	case "ssh":
		if u.Host == "" || (u.User != nil && u.User.Username() == "") {
			return errors.New("SSH publication target is invalid")
		}
		if u.User != nil {
			if _, hasPassword := u.User.Password(); hasPassword {
				return errors.New("SSH publication target must not contain a password")
			}
		}
	case "file":
		if u.Path == "" || u.Host != "" {
			return errors.New("file publication target must be a local absolute URL")
		}
	default:
		return errors.New("publication target must use HTTPS, SSH, scp-style SSH, or a local file path")
	}
	return nil
}

func isolatedPublishGitDir(ctx context.Context, root string) (string, error) {
	dir, err := os.MkdirTemp("", "workbench-publish-git-")
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "git", "init", "--bare", "--quiet", dir)
	configureChildProcess(cmd, false)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("create isolated publication repository: %s", strings.TrimSpace(string(out)))
	}
	objects, err := runGitLimited(ctx, root, 64<<10, "rev-parse", "--git-path", "objects")
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	objects = strings.TrimSpace(objects)
	if !filepath.IsAbs(objects) {
		objects = filepath.Join(root, objects)
	}
	objects, err = filepath.Abs(objects)
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "objects", "info", "alternates"), []byte(objects+"\n"), 0o600); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func remoteBranchSHA(ctx context.Context, gitDir, remoteURL, ref string) (string, bool, error) {
	cmd := publishGitCommand(ctx, gitDir, "ls-remote", "--heads", remoteURL, ref)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("inspect publication target: %s", strings.TrimSpace(out.String()))
	}
	line := strings.TrimSpace(out.String())
	if line == "" {
		return "", false, nil
	}
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[1] != ref {
		return "", false, errors.New("publication target returned an unexpected branch response")
	}
	return fields[0], true, nil
}

func publishGitCommand(ctx context.Context, gitDir string, args ...string) *exec.Cmd {
	cmdArgs := append([]string{"--git-dir", gitDir}, args...)
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "PAGER=cat", "GIT_PAGER=cat")
	configureChildProcess(cmd, false)
	return cmd
}
