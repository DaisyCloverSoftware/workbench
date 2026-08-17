package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateControlOutboxRedactsImplicitRunnerHostPaths(t *testing.T) {
	out := privateControlOutbox{
		Version: 1,
		ID:      "redaction-12345678",
		Action:  "read_file",
		Status:  "completed",
		Result: map[string]any{
			"project_path": "/srv/workbench/projects/garage",
			"path":         "README.md",
			"content":      "keep this relative file result",
			"capsule": map[string]any{
				"project": "/srv/workbench/projects/garage",
				"state":   "keep this context",
			},
			"memory": map[string]any{
				"project": `C:\\workbench\\projects\\garage`,
				"content": "keep this memory",
			},
			"project": "runner://garage",
		},
	}

	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, forbidden := range []string{"/srv/workbench/projects/garage", `C:\\workbench\\projects\\garage`, `"project_path"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("private control result leaked host path metadata %q: %s", forbidden, text)
		}
	}
	for _, want := range []string{"README.md", "keep this relative file result", "keep this context", "keep this memory", "runner://garage"} {
		if !strings.Contains(text, want) {
			t.Fatalf("private control redaction removed useful result %q: %s", want, text)
		}
	}
}

func TestLooksLikeAbsoluteHostPathIsCrossPlatform(t *testing.T) {
	for _, value := range []string{"/srv/workbench/projects/repo", `C:\\workbench\\projects\\repo`, `D:/work/repo`, `\\\\server\\share\\repo`} {
		if !looksLikeAbsoluteHostPath(value) {
			t.Fatalf("expected absolute host path %q", value)
		}
	}
	for _, value := range []string{"runner://garage", "README.md", "docs/guide.md", "project-name"} {
		if looksLikeAbsoluteHostPath(value) {
			t.Fatalf("unexpected host path classification for %q", value)
		}
	}
}
