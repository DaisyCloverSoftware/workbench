package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	runnerProjectDiscoveryDepth = 4
	runnerProjectDiscoveryDirs  = 5000
	runnerProjectDiscoveryRepos = 500
)

type runnerDiscoveryDir struct {
	path  string
	depth int
}

// discoverRunnerProjects scans the operator-authorised roots rather than only
// their immediate children. Real installations commonly group repositories one
// level deeper (for example by organisation/product), and a direct-child-only
// picker makes perfectly valid active projects disappear from Workbench.
//
// The scan is deliberately bounded, never follows a symlink outside an
// authorised root, stops descending once a Git toplevel is found, and skips
// generated/cache directories.
func discoverRunnerProjects(ctx context.Context) ([]discoveredRunnerProject, error) {
	specs, err := runnerRootSpecs()
	if err != nil {
		return nil, err
	}
	discovered := make([]discoveredRunnerProject, 0, 32)
	seenRepos := map[string]bool{}
	visited := 0
	for _, spec := range specs {
		rootProjects, count, scanErr := discoverRunnerProjectsInRoot(ctx, spec, runnerProjectDiscoveryDirs-visited)
		visited += count
		if scanErr != nil {
			return nil, scanErr
		}
		for _, project := range rootProjects {
			if seenRepos[project.path] {
				continue
			}
			seenRepos[project.path] = true
			discovered = append(discovered, project)
			if len(discovered) >= runnerProjectDiscoveryRepos {
				return discovered, nil
			}
		}
		if visited >= runnerProjectDiscoveryDirs {
			break
		}
	}
	return discovered, nil
}

func discoverRunnerProjectsInRoot(ctx context.Context, spec runnerRootSpec, remaining int) ([]discoveredRunnerProject, int, error) {
	if remaining <= 0 {
		return nil, 0, nil
	}
	queue := []runnerDiscoveryDir{{path: spec.Path, depth: 0}}
	seenDirs := map[string]bool{}
	projects := make([]discoveredRunnerProject, 0, 16)
	visited := 0

	for len(queue) > 0 && visited < remaining && len(projects) < runnerProjectDiscoveryRepos {
		select {
		case <-ctx.Done():
			return nil, visited, ctx.Err()
		default:
		}

		current := queue[0]
		queue = queue[1:]
		resolved, err := canonicalRunnerDirectory(current.path)
		if err != nil || !withinRoot(spec.Path, resolved) || seenDirs[resolved] {
			continue
		}
		seenDirs[resolved] = true
		visited++

		if current.depth > 0 {
			if ok := runnerDirectoryIsGitToplevel(ctx, resolved); ok {
				name := filepath.Base(resolved)
				if _, nameErr := validateRunnerProjectName(name); nameErr == nil {
					projects = append(projects, discoveredRunnerProject{name: name, rootNumber: spec.Number, path: resolved})
				}
				// Do not walk generated/source trees inside an already discovered
				// repository. Independent project repositories are expected below
				// authorised organisation/workspace containers, not inside another
				// repository's working tree.
				continue
			}
		}
		if current.depth >= runnerProjectDiscoveryDepth {
			continue
		}

		entries, readErr := os.ReadDir(resolved)
		if readErr != nil {
			if current.depth == 0 {
				return nil, visited, readErr
			}
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if skipRunnerDiscoveryDirectory(name) {
				continue
			}
			candidate := filepath.Join(resolved, name)
			if entry.IsDir() {
				queue = append(queue, runnerDiscoveryDir{path: candidate, depth: current.depth + 1})
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				// canonicalRunnerDirectory resolves the link and the next loop
				// proves containment before entering it.
				if target, targetErr := canonicalRunnerDirectory(candidate); targetErr == nil && withinRoot(spec.Path, target) {
					queue = append(queue, runnerDiscoveryDir{path: target, depth: current.depth + 1})
				}
			}
		}
	}
	return projects, visited, nil
}

func runnerDirectoryIsGitToplevel(ctx context.Context, dir string) bool {
	gitRoot, err := runGitLimited(ctx, dir, 4096, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}
	canonical, err := canonicalRunnerDirectory(strings.TrimSpace(gitRoot))
	return err == nil && filepath.Clean(canonical) == filepath.Clean(dir)
}

func skipRunnerDiscoveryDirectory(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "", ".git", ".cache", ".worktrees", ".worktree", "node_modules", "vendor", "dist", "build", "out", "target", ".venv", "venv", "__pycache__", ".next", ".nuxt", "coverage":
		return true
	default:
		return false
	}
}

// runnerProjectMatchesByName locates an existing repository with the requested
// basename under one authorised root. It uses the same bounded discovery rules
// as the picker so a runner:// reference returned by discovery is resolvable by
// later safe-hands/task operations even when the repo is nested.
func runnerProjectMatchesByName(root, name string) ([]string, error) {
	name, err := validateRunnerProjectName(name)
	if err != nil {
		return nil, err
	}
	canonicalRoot, err := canonicalRunnerDirectory(root)
	if err != nil {
		return nil, err
	}
	queue := []runnerDiscoveryDir{{path: canonicalRoot, depth: 0}}
	seen := map[string]bool{}
	matches := []string{}
	visited := 0

	for len(queue) > 0 && visited < runnerProjectDiscoveryDirs {
		current := queue[0]
		queue = queue[1:]
		resolved, resolveErr := canonicalRunnerDirectory(current.path)
		if resolveErr != nil || !withinRoot(canonicalRoot, resolved) || seen[resolved] {
			continue
		}
		seen[resolved] = true
		visited++

		if current.depth > 0 && filepath.Base(resolved) == name {
			ctx, cancel := context.WithCancel(context.Background())
			ok := runnerDirectoryIsGitToplevel(ctx, resolved)
			cancel()
			if ok {
				matches = append(matches, resolved)
				continue
			}
		}
		if current.depth >= runnerProjectDiscoveryDepth {
			continue
		}
		entries, readErr := os.ReadDir(resolved)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if skipRunnerDiscoveryDirectory(entry.Name()) {
				continue
			}
			candidate := filepath.Join(resolved, entry.Name())
			if entry.IsDir() {
				queue = append(queue, runnerDiscoveryDir{path: candidate, depth: current.depth + 1})
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if target, targetErr := canonicalRunnerDirectory(candidate); targetErr == nil && withinRoot(canonicalRoot, target) {
					queue = append(queue, runnerDiscoveryDir{path: target, depth: current.depth + 1})
				}
			}
		}
	}
	sort.Strings(matches)
	return matches, nil
}

func uniqueRunnerProjectMatch(matches []string, notFound, ambiguous string) (string, error) {
	switch len(matches) {
	case 0:
		return "", errors.New(notFound)
	case 1:
		return matches[0], nil
	default:
		return "", errors.New(ambiguous)
	}
}
