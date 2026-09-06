//go:build windows

package runnerdiag

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerDiagnosticWindowsPaths(t *testing.T) {
	for _, p := range []string{`C:\actions-runner`, `D:\tools\runner`} {
		if !safeAbsolute(p) {
			t.Fatal(p)
		}
	}
	for _, p := range []string{`C:relative`, `\\server\runner`, `\\?\C:\runner`, `C:\a\..\b`, `C:\a\file:ads`, `C:\repo\.git\runner`, `C:\bad.\runner`, `%HOME%\runner`} {
		if safeAbsolute(p) {
			t.Fatal(p)
		}
	}
	for _, p := range []string{`C:\actions-runner\bin\RunnerService.exe`, `"C:\runner with space\bin\RunnerService.exe"`} {
		if installationFromImage(p) == "" {
			t.Fatal(p)
		}
	}
	for _, p := range []string{`"C:\runner\bin\RunnerService.exe" --token secret`, `C:\runner\bin\evil.exe`, `C:\runner\bin\RunnerService.exe --arg`} {
		if installationFromImage(p) != "" {
			t.Fatal("unsafe service command accepted")
		}
	}
}
func TestRunnerDiagnosticWindowsFixtureReadOnly(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "bin"), 0700); e != nil {
		t.Fatal(e)
	}
	fixtures := map[string][]byte{
		".runner":      []byte(`{"agentId":777,"gitHubUrl":"https://github.com/example/fixture","workFolder":"_work"}`),
		".credentials": []byte("NEVER_READ_SENTINEL"), ".service": []byte("actions.runner.example.fixture"),
		"config.cmd": []byte("not executed"), "bin/Runner.Listener.exe": []byte("not executed"), "bin/RunnerService.exe": []byte("not executed"),
	}
	for name, data := range fixtures {
		if e := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	i, status := inspectRoot(candidate{root, "fixture"})
	if status != "ok" || i.Config != "readable" || i.AgentID != 777 || i.LocalState != "configured_locally_registration_unverified" {
		t.Fatalf("status=%s config=%s state=%s", status, i.Config, i.LocalState)
	}
	for name, data := range fixtures {
		got, e := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if e != nil || !bytes.Equal(data, got) {
			t.Fatal("fixture modified")
		}
	}
	r := newReport()
	r.Installations = []Installation{i}
	out, e := encodeReport(r)
	if e != nil || strings.Contains(out, root) || strings.Contains(out, "NEVER_READ_SENTINEL") {
		t.Fatal("sensitive output")
	}
	if _, state := fixedFile(root, ".credentials", true); state != "not_allowed" {
		t.Fatal("credential path exposed")
	}
	if _, state := fixedFile(root, "config.cmd", true); state != "not_allowed" {
		t.Fatal("command file readable")
	}
}
func TestRunnerDiagnosticWindowsAbsentAndGitExcluded(t *testing.T) {
	root := t.TempDir()
	if _, state := inspectRoot(candidate{filepath.Join(root, "missing"), "fixture"}); state != "absent" {
		t.Fatal(state)
	}
	if e := os.Mkdir(filepath.Join(root, ".git"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, state := inspectRoot(candidate{root, "fixture"}); state != "git_worktree_excluded" {
		t.Fatal(state)
	}
}
func TestRunnerDiagnosticWindowsRejectsMetadataHardlinks(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "metadata")
	if e := os.WriteFile(original, []byte(`{"agentId":1}`), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Link(original, filepath.Join(root, ".runner")); e != nil {
		t.Skip("hardlinks unavailable")
	}
	if _, state := fixedFile(root, ".runner", true); state != "unsafe_file" {
		t.Fatal(state)
	}
}
func TestRunnerDiagnosticWindowsNativeQuerySmoke(t *testing.T) {
	// Query-only smoke of the hosted Windows machine; do not print its inventory.
	r, e := Inspect()
	if e != nil {
		t.Fatal("inspection failed")
	}
	if !r.ReadOnly || r.MachineWideAbsenceEstablished || r.UnrealProof != "not_executed" {
		t.Fatal("false acceptance")
	}
	if _, e := encodeReport(r); e != nil {
		t.Fatal("invalid bounded report")
	}
}
