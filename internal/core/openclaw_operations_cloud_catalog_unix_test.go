//go:build !windows

package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOperationsCloudCatalogRestoresOpenClawSiblingNode(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}

	node := filepath.Join(bin, "node")
	nodeBody := `#!/bin/sh
set -eu
shift
if [ "${1:-}" = "models" ] && [ "${2:-}" = "status" ]; then
  echo '{"resolvedDefault":"openai/gpt-5.3-codex-spark"}'
  exit 0
fi
if [ "${1:-}" = "models" ] && [ "${2:-}" = "list" ]; then
  echo '{"models":[{"key":"openai/gpt-5.3-codex-spark","provider":"openai","available":true},{"key":"anthropic/claude-sonnet","provider":"anthropic","available":true}]}'
  exit 0
fi
exit 2
`
	if err := os.WriteFile(node, []byte(nodeBody), 0o755); err != nil {
		t.Fatal(err)
	}
	openclaw := filepath.Join(bin, "openclaw")
	if err := os.WriteFile(openclaw, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	catalog, err := discoverOpenClawOperationsCloudCatalog(context.Background(), openclaw)
	if err != nil {
		t.Fatalf("service-safe OpenClaw cloud discovery failed: %v", err)
	}
	if catalog.DefaultModel != "openai/gpt-5.3-codex-spark" {
		t.Fatalf("default=%q", catalog.DefaultModel)
	}
	fallback, ok := preferredOpenClawOperationsCloudFallback(catalog, "openai")
	if !ok || fallback.Key != "anthropic/claude-sonnet" {
		t.Fatalf("fallback=%+v ok=%t", fallback, ok)
	}
}
