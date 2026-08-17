package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareReleaseUpdatesEveryVersionContractAndChangelog(t *testing.T) {
	root := t.TempDir()
	writeFixture := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFixture(".codex-plugin/plugin.json", "{\n  \"version\": \"0.9.5\"\n}\n")
	writeFixture("cmd/workbench/main_windows.go", "package main\nconst appVersion = \"0.9.5\"\n")
	writeFixture("cmd/workbench-runner/main.go", "package main\nconst runnerVersion = \"0.9.5\"\n")
	writeFixture("cmd/workbench-server/main.go", "package main\nconst serverVersion = \"0.9.5\"\n")
	writeFixture("cmd/workbench-relay/main.go", "package main\nconst relayVersion = \"0.9.5\"\n")
	writeFixture("internal/core/version.go", "package core\nconst Version = \"0.9.5\"\n")
	writeFixture("internal/mcp/server.go", "package mcp\nvar info = map[string]string{\"version\": \"0.9.5\"}\n")
	writeFixture("CHANGELOG.md", "# Changelog\n\n## 0.9.5 — 2026-08-17\n\n- Previous release.\n")

	req := releaseRequest{Version: "0.9.6", Date: "2026-08-17", Notes: []string{"First new capability.", "Second verified fix."}}
	if err := prepareRelease(root, req); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		".codex-plugin/plugin.json",
		"cmd/workbench/main_windows.go",
		"cmd/workbench-runner/main.go",
		"cmd/workbench-server/main.go",
		"cmd/workbench-relay/main.go",
		"internal/core/version.go",
		"internal/mcp/server.go",
	} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if !strings.Contains(text, "0.9.6") || strings.Contains(text, "0.9.5") {
			t.Fatalf("%s was not coordinated to 0.9.6: %q", path, text)
		}
	}
	changelog, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := "# Changelog\n\n## 0.9.6 — 2026-08-17\n\n- First new capability.\n- Second verified fix.\n\n## 0.9.5"
	if !strings.HasPrefix(string(changelog), wantPrefix) {
		t.Fatalf("unexpected changelog:\n%s", changelog)
	}
}

func TestPrepareReleaseValidatesAllTargetsBeforeWriting(t *testing.T) {
	root := t.TempDir()
	paths := map[string]string{
		".codex-plugin/plugin.json":         "{\"version\": \"0.9.5\"}",
		"cmd/workbench/main_windows.go":     "const appVersion = \"0.9.5\"",
		"cmd/workbench-runner/main.go":      "const runnerVersion = \"0.9.5\"",
		"cmd/workbench-server/main.go":      "const serverVersion = \"0.9.5\"",
		"cmd/workbench-relay/main.go":       "const relayVersion = \"WRONG\"",
		"internal/core/version.go":          "const Version = \"0.9.5\"",
		"internal/mcp/server.go":            "{\"version\": \"0.9.5\"}",
		"CHANGELOG.md":                      "# Changelog\n\n## 0.9.5\n",
	}
	for path, content := range paths {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := os.ReadFile(filepath.Join(root, "internal", "core", "version.go"))
	err := prepareRelease(root, releaseRequest{Version: "0.9.6", Date: "2026-08-17", Notes: []string{"Test."}})
	if err == nil {
		t.Fatal("expected malformed version contract to fail")
	}
	after, _ := os.ReadFile(filepath.Join(root, "internal", "core", "version.go"))
	if string(after) != string(before) {
		t.Fatalf("canonical version changed despite preflight failure: before=%q after=%q", before, after)
	}
}

func TestVersionGreater(t *testing.T) {
	cases := []struct {
		next, current string
		want          bool
	}{
		{"0.9.6", "0.9.5", true},
		{"0.10.0", "0.9.99", true},
		{"1.0.0", "0.99.99", true},
		{"0.9.5", "0.9.5", false},
		{"0.9.4", "0.9.5", false},
	}
	for _, tc := range cases {
		if got := versionGreater(tc.next, tc.current); got != tc.want {
			t.Fatalf("versionGreater(%q,%q)=%v want %v", tc.next, tc.current, got, tc.want)
		}
	}
}
