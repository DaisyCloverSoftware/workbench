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
	if path, err := exec.LookPath("openclaw"); err == nil {
		return path, true
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", false
	}

	names := []string{"openclaw"}
	if runtime.GOOS == "windows" {
		names = []string{"openclaw.exe", "openclaw.cmd", "openclaw.bat", "openclaw"}
	}

	var candidates []string
	for _, name := range names {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, "bin", name),
			filepath.Join(home, ".npm-global", "bin", name),
			filepath.Join(home, ".npm", "bin", name),
			filepath.Join(home, ".local", "share", "pnpm", name),
			filepath.Join(home, ".volta", "bin", name),
			filepath.Join(home, ".bun", "bin", name),
			filepath.Join(home, ".openclaw", "bin", name),
		)
		for _, pattern := range []string{
			filepath.Join(home, ".nvm", "versions", "node", "*", "bin", name),
			filepath.Join(home, ".local", "share", "fnm", "node-versions", "*", "installation", "bin", name),
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
