package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestBuildOutboxPublicOmitsPrivateMetadataAndReportContent(t *testing.T) {
	task := core.Task{
		ID:                "task-123",
		Status:            core.TaskCompleted,
		ConsumesWork:      true,
		Output:            "private report body",
		Error:             "private error body",
		AttentionQuestion: "private attention question",
		UpdatedAt:         time.Unix(123, 0).UTC(),
	}
	out := buildOutbox("relay-12345678", task, "status", true)
	if out.Status != core.TaskCompleted {
		t.Fatalf("status = %q", out.Status)
	}
	if out.WorkbenchTask != "" || out.ConsumesWork {
		t.Fatalf("public outbox leaked private execution metadata: %#v", out)
	}
	if out.Report != "" || out.Error != "" || out.Attention != "" || out.DetailWithheld {
		t.Fatalf("public outbox leaked report metadata/content: %#v", out)
	}
}

func TestBuildOutboxPrivateIncludesReportContent(t *testing.T) {
	task := core.Task{
		ID:                "task-123",
		Status:            core.TaskNeedsAttention,
		ConsumesWork:      true,
		Output:            "worker report",
		Error:             "worker error",
		AttentionQuestion: "approve this choice?",
		UpdatedAt:         time.Unix(456, 0).UTC(),
	}
	out := buildOutbox("relay-12345678", task, "report", false)
	if out.WorkbenchTask != task.ID || !out.ConsumesWork {
		t.Fatalf("private outbox omitted execution metadata: %#v", out)
	}
	if out.Report != task.Output || out.Error != task.Error || out.Attention != task.AttentionQuestion || out.DetailWithheld {
		t.Fatalf("private outbox did not include report content: %#v", out)
	}
}

func TestBuildOutboxWithholdsSecretLikeReport(t *testing.T) {
	task := core.Task{
		ID:        "task-123",
		Status:    core.TaskCompleted,
		Output:    "credential accidentally echoed: " + "sk-" + "proj-" + strings.Repeat("x", 48),
		UpdatedAt: time.Unix(789, 0).UTC(),
	}
	out := buildOutbox("relay-12345678", task, "report", false)
	if !out.DetailWithheld {
		t.Fatalf("expected secret-like detail to be withheld: %#v", out)
	}
	if out.Report != "" || out.Error != "" || out.Attention != "" {
		t.Fatalf("withheld outbox still contains detail: %#v", out)
	}
}

func TestTextDigestNormalizesWhitespace(t *testing.T) {
	if textDigest("  yes  ") != textDigest("yes") {
		t.Fatal("digest should ignore surrounding whitespace")
	}
	if textDigest("yes") == textDigest("no") {
		t.Fatal("different answers should have different digests")
	}
}

func TestPublishOutboxIsIdempotent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "remote.git")
	seed := filepath.Join(root, "seed")
	relay := filepath.Join(root, "relay")

	runGit(t, "init", "--bare", bare)
	runGit(t, "init", "-b", "main", seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("relay transport fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", seed, "add", "README.md")
	runGit(t, "-C", seed, "-c", "user.name=Workbench Test", "-c", "user.email=workbench-test@example.invalid", "commit", "-m", "seed")
	runGit(t, "-C", seed, "remote", "add", "origin", bare)
	runGit(t, "-C", seed, "push", "origin", "main")
	runGit(t, "--git-dir", bare, "symbolic-ref", "HEAD", "refs/heads/main")
	runGit(t, "clone", "--quiet", bare, relay)

	files := map[string][]byte{
		"relay/outbox/relay-12345678.json": []byte("{\"version\":1,\"id\":\"relay-12345678\",\"status\":\"completed\"}\n"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := publishOutbox(ctx, relay, "origin", "main", files); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", relay, "fetch", "--quiet", "origin", "main")
	got := runGit(t, "-C", relay, "show", "origin/main:relay/outbox/relay-12345678.json")
	if got != string(files["relay/outbox/relay-12345678.json"]) {
		t.Fatalf("outbox content mismatch: %q", got)
	}
	countBefore := strings.TrimSpace(runGit(t, "-C", relay, "rev-list", "--count", "origin/main"))
	if err := publishOutbox(ctx, relay, "origin", "main", files); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", relay, "fetch", "--quiet", "origin", "main")
	countAfter := strings.TrimSpace(runGit(t, "-C", relay, "rev-list", "--count", "origin/main"))
	if countBefore != countAfter {
		t.Fatalf("idempotent publish created another commit: before=%s after=%s", countBefore, countAfter)
	}
}

func runGit(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
