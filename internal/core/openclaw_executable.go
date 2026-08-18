package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// findOpenClawExecutable keeps Workbench services from depending on an
// interactive shell's PATH. OpenClaw is commonly installed into a user-level
// Node/pnpm/Volta/NVM directory that systemd --user does not inherit even though
// `openclaw` works in the user's terminal.
func findOpenClawExecutable() (string, bool) {
	return findUserExecutable("openclaw")
}

func findUserExecutable(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, true
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false
	}

	names := []string{name}
	if runtime.GOOS == "windows" {
		names = []string{name + ".exe", name + ".cmd", name + ".bat", name}
	}

	var candidates []string
	for _, executableName := range names {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", executableName),
			filepath.Join(home, "bin", executableName),
			filepath.Join(home, ".npm-global", "bin", executableName),
			filepath.Join(home, ".npm", "bin", executableName),
			filepath.Join(home, ".local", "share", "pnpm", executableName),
			filepath.Join(home, ".volta", "bin", executableName),
			filepath.Join(home, ".bun", "bin", executableName),
			filepath.Join(home, ".openclaw", "bin", executableName),
		)
		for _, pattern := range []string{
			filepath.Join(home, ".nvm", "versions", "node", "*", "bin", executableName),
			filepath.Join(home, ".local", "share", "fnm", "node-versions", "*", "installation", "bin", executableName),
		} {
			matches, _ := filepath.Glob(pattern)
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			candidates = append(candidates, matches...)
		}
	}

	for _, candidate := range candidates {
		if executableFile(candidate) {
			return candidate, true
		}
	}
	return "", false
}

// environmentForUserExecutable makes an absolute user-level CLI genuinely
// runnable from a service, not merely discoverable. npm/pnpm/NVM installs often
// make `openclaw` a script whose shebang is `#!/usr/bin/env node`; systemd may
// see the OpenClaw path while still being unable to find that Node interpreter.
// Keep the existing environment intact and prepend only bounded executable /
// interpreter directories to PATH.
func environmentForUserExecutable(executable string) []string {
	env := os.Environ()
	var dirs []string
	addDir := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" || dir == "." {
			return
		}
		for _, existing := range dirs {
			if sameEnvironmentPath(existing, dir) {
				return
			}
		}
		dirs = append(dirs, dir)
	}

	if executable = strings.TrimSpace(executable); executable != "" {
		if abs, err := filepath.Abs(executable); err == nil {
			addDir(filepath.Dir(abs))
			if resolved, err := filepath.EvalSymlinks(abs); err == nil {
				addDir(filepath.Dir(resolved))
			}
		}
	}
	if node, ok := findUserExecutable("node"); ok {
		if abs, err := filepath.Abs(node); err == nil {
			addDir(filepath.Dir(abs))
		}
	}

	oldPath := os.Getenv("PATH")
	parts := append([]string(nil), dirs...)
	for _, dir := range filepath.SplitList(oldPath) {
		add := true
		for _, existing := range parts {
			if sameEnvironmentPath(existing, dir) {
				add = false
				break
			}
		if add && strings.TrimSpace(dir) != "" {
			parts = append(parts, dir)
		}
	}
	newPath := strings.Join(parts, string(os.PathListSeparator))

	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		key, _, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "PATH") {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, "PATH="+newPath)
	return out
}

func sameEnvironmentPath(a, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func executableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}
