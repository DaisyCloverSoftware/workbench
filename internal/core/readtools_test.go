package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadProjectFileAndSearchStayInsideProject(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "package demo\n\nfunc Answer() int {\n\treturn 42\n}\n"
	if err := os.WriteFile(filepath.Join(root, "src", "answer.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadProjectFile(root, "src/answer.go", 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "func Answer") || !strings.Contains(got, "return 42") {
		t.Fatalf("unexpected read output: %s", got)
	}

	hits, err := SearchProjectText(root, "answer", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Path != "src/answer.go" || hits[0].Line != 3 {
		t.Fatalf("unexpected hits: %#v", hits)
	}

	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outside, []byte("not yours"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProjectFile(root, "../outside.txt", 0, 0); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestModelFacingReadsRefuseSecretsAndGeneratedTrees(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("API_KEY=supersecretvalue12345\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.txt"), []byte("token=abcdefghijk123456789\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.txt"), []byte("hello Workbench\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "pkg", "noise.js"), []byte("Workbench\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadProjectFile(root, ".env", 0, 0); err == nil {
		t.Fatal("expected .env refusal")
	}
	if _, err := ReadProjectFile(root, "config.txt", 0, 0); err == nil {
		t.Fatal("expected probable secret content refusal")
	}
	files, err := ListProjectFiles(root, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(files, "\n")
	if strings.Contains(joined, ".env") || strings.Contains(joined, "node_modules") {
		t.Fatalf("sensitive/generated paths leaked from listing: %s", joined)
	}
	if !strings.Contains(joined, "main.txt") {
		t.Fatalf("normal source file missing: %s", joined)
	}

	hits, err := SearchProjectText(root, "Workbench", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Path != "main.txt" {
		t.Fatalf("search should skip generated/sensitive content: %#v", hits)
	}
}
