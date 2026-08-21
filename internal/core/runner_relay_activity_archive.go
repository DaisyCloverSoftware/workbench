package core

import (
	"errors"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const maxRelayActivityFiles = 600

var relayActivityPrefixes = []string{
	"relay/control",
	"relay/control-outbox",
	"relay/inbox",
	"relay/outbox",
}

// readRunnerRelayActivityArchive builds a bounded view of the private relay.
// The relay is an append-only transport, so archiving its complete history on
// every desktop refresh makes dashboard cost grow forever. Keep every pending
// request, then add the newest files from the activity window together with
// their request/result counterpart. This makes the live inventory proportional
// to current work rather than repository age.
func readRunnerRelayActivityArchive(repo string) ([]byte, error) {
	const ref = "origin/main"
	paths, err := selectRunnerRelayActivityPaths(repo, ref)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	args := []string{"-C", repo, "archive", "--format=tar", ref, "--"}
	args = append(args, paths...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, errors.New("private Workbench relay activity is unavailable")
	}
	if len(out) > maxRelayActivityArchive {
		return nil, errors.New("private Workbench relay activity exceeded bounds")
	}
	return out, nil
}

func selectRunnerRelayActivityPaths(repo, ref string) ([]string, error) {
	all, err := gitRelayActivityPathList(repo, ref)
	if err != nil {
		return nil, err
	}
	controls, controlResults := map[string]string{}, map[string]string{}
	autonomous, autonomousResults := map[string]string{}, map[string]string{}
	for _, path := range all {
		id := strings.TrimSuffix(filepath.Base(path), ".json")
		if id == "" || !strings.HasSuffix(path, ".json") {
			continue
		}
		switch {
		case strings.HasPrefix(path, "relay/control/"):
			controls[id] = path
		case strings.HasPrefix(path, "relay/control-outbox/"):
			controlResults[id] = path
		case strings.HasPrefix(path, "relay/inbox/"):
			autonomous[id] = path
		case strings.HasPrefix(path, "relay/outbox/"):
			autonomousResults[id] = path
		}
	}

	selected := map[string]bool{}
	add := func(path string) bool {
		if path == "" || selected[path] {
			return true
		}
		if len(selected) >= maxRelayActivityFiles {
			return false
		}
		selected[path] = true
		return true
	}
	addPair := func(path string) bool {
		if !add(path) {
			return false
		}
		id := strings.TrimSuffix(filepath.Base(path), ".json")
		switch {
		case strings.HasPrefix(path, "relay/control/"):
			return add(controlResults[id])
		case strings.HasPrefix(path, "relay/control-outbox/"):
			return add(controls[id])
		case strings.HasPrefix(path, "relay/inbox/"):
			return add(autonomousResults[id])
		case strings.HasPrefix(path, "relay/outbox/"):
			return add(autonomous[id])
		}
		return true
	}

	// Pending work is authoritative live state and must never be displaced by
	// historical activity.
	for id, path := range controls {
		if controlResults[id] == "" && !add(path) {
			return nil, errors.New("private Workbench relay has too many pending controls")
		}
	}
	for id, path := range autonomous {
		if autonomousResults[id] == "" && !add(path) {
			return nil, errors.New("private Workbench relay has too many pending tasks")
		}
	}

	recent, err := gitRecentRelayActivityPaths(repo, ref)
	if err != nil {
		return nil, err
	}
	for _, path := range recent {
		if len(selected) >= maxRelayActivityFiles {
			break
		}
		_ = addPair(path)
	}

	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func gitRelayActivityPathList(repo, ref string) ([]string, error) {
	args := []string{"-C", repo, "ls-tree", "-r", "--name-only", ref, "--"}
	args = append(args, relayActivityPrefixes...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, errors.New("private Workbench relay activity is unavailable")
	}
	return relayActivityLines(out), nil
}

func gitRecentRelayActivityPaths(repo, ref string) ([]string, error) {
	args := []string{"-C", repo, "log", "--since=4 hours ago", "--pretty=format:", "--name-only", ref, "--"}
	args = append(args, relayActivityPrefixes...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, errors.New("private Workbench relay activity is unavailable")
	}
	return relayActivityLines(out), nil
}

func relayActivityLines(raw []byte) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 128)
	for _, line := range strings.Split(string(raw), "\n") {
		line = filepath.ToSlash(strings.TrimSpace(line))
		if line == "" || seen[line] || !strings.HasSuffix(line, ".json") {
			continue
		}
		valid := false
		for _, prefix := range relayActivityPrefixes {
			if strings.HasPrefix(line, prefix+"/") {
				valid = true
				break
			}
		}
		if !valid {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	return out
}
