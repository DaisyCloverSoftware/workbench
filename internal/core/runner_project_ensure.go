package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const runnerProjectCloneTimeout = 10 * time.Minute

type runnerProjectCloneFunc func(context.Context, string, string) error

// EnsureRunnerGitHubProject makes one GitHub repository available as a normal
// runner:// project without accepting an arbitrary remote URL. Existing runner
// projects are returned untouched. Missing repositories are cloned into the
// first operator-authorised project root using a temporary same-filesystem
// directory and atomic rename, so a failed clone never leaves a half-project
// that the desktop can mistake for usable work.
func EnsureRunnerGitHubProject(ctx context.Context, repository string) (RunnerProjectInfo, bool, error) {
	return ensureRunnerGitHubProject(ctx, repository, cloneGitHubRepository)
}

func ensureRunnerGitHubProject(ctx context.Context, repository string, clone runnerProjectCloneFunc) (RunnerProjectInfo, bool, error) {
	owner, name, err := validateGitHubRepositorySlug(repository)
	if err != nil {
		return RunnerProjectInfo{}, false, err
	}
	projects, err := listRunnerProjects(ctx)
	if err != nil {
		return RunnerProjectInfo{}, false, err
	}
	matches := make([]RunnerProjectInfo, 0, 1)
	for _, project := range projects {
		if strings.EqualFold(project.Name, name) {
			matches = append(matches, project)
		}
	}
	if len(matches) == 1 {
		return matches[0], false, nil
	}
	if len(matches) > 1 {
		return RunnerProjectInfo{}, false, fmt.Errorf("runner already contains multiple projects named %q; choose the existing scoped project explicitly", name)
	}

	roots, err := runnerRoots()
	if err != nil {
		return RunnerProjectInfo{}, false, err
	}
	root := roots[0]
	target := filepath.Join(root, name)
	if _, statErr := os.Lstat(target); statErr == nil {
		return RunnerProjectInfo{}, false, errors.New("runner destination exists but is not a discovered Git repository")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return RunnerProjectInfo{}, false, errors.New("runner destination could not be inspected")
	}

	tmp, err := os.MkdirTemp(root, ".workbench-clone-"+name+"-")
	if err != nil {
		return RunnerProjectInfo{}, false, errors.New("runner could not create a temporary clone directory")
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tmp)
		}
	}()

	cloneCtx, cancel := context.WithTimeout(ctx, runnerProjectCloneTimeout)
	cloneErr := clone(cloneCtx, owner+"/"+name, tmp)
	cancel()
	if cloneErr != nil {
		return RunnerProjectInfo{}, false, errors.New("GitHub project could not be cloned with the runner's existing non-interactive GitHub credentials")
	}
	gitRoot, err := runGitLimited(ctx, tmp, 4096, "rev-parse", "--show-toplevel")
	if err != nil {
		return RunnerProjectInfo{}, false, errors.New("cloned project did not verify as a Git repository")
	}
	verified, err := canonicalRunnerDirectory(strings.TrimSpace(gitRoot))
	if err != nil || filepath.Clean(verified) != filepath.Clean(tmp) || !withinRoot(root, verified) {
		return RunnerProjectInfo{}, false, errors.New("cloned project failed runner-root containment verification")
	}
	if err := os.Rename(tmp, target); err != nil {
		return RunnerProjectInfo{}, false, errors.New("cloned project could not be installed into the runner project root")
	}
	cleanup = false

	projects, err = listRunnerProjects(ctx)
	if err != nil {
		return RunnerProjectInfo{}, true, err
	}
	for _, project := range projects {
		if strings.EqualFold(project.Name, name) {
			return project, true, nil
		}
	}
	return RunnerProjectInfo{}, true, errors.New("new GitHub project was cloned but did not appear in runner discovery")
}

func validateGitHubRepositorySlug(repository string) (owner, name string, err error) {
	repository = strings.TrimSpace(repository)
	if len(repository) < 3 || len(repository) > 200 || strings.Count(repository, "/") != 1 {
		return "", "", errors.New("repository must be one GitHub owner/name slug")
	}
	parts := strings.SplitN(repository, "/", 2)
	owner, name = parts[0], parts[1]
	if !validGitHubSlugPart(owner) || !validGitHubSlugPart(name) || strings.HasSuffix(name, ".git") {
		return "", "", errors.New("repository must be one GitHub owner/name slug")
	}
	if _, err := validateRunnerProjectName(name); err != nil {
		return "", "", errors.New("GitHub repository name is not safe as a runner project name")
	}
	return owner, name, nil
}

func validGitHubSlugPart(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func cloneGitHubRepository(ctx context.Context, repository, target string) error {
	if gh, err := exec.LookPath("gh"); err == nil {
		cmd := exec.CommandContext(ctx, gh, "repo", "clone", repository, target, "--", "--quiet")
		if err := cmd.Run(); err == nil {
			return nil
		}
		// gh may be installed but unauthenticated for the service account. The
		// fixed-host SSH fallback reuses normal non-interactive Git credentials.
		_ = os.RemoveAll(target)
		if err := os.MkdirAll(target, 0o700); err != nil {
			return err
		}
	}
	remote := "git@github.com:" + repository + ".git"
	cmd := exec.CommandContext(ctx, "git", "clone", "--quiet", "--", remote, target)
	return cmd.Run()
}
