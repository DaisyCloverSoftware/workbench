package main

import (
	"path/filepath"
	"testing"
)

func TestPrivateUpdateSourceDirUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WORKBENCH_SOURCE_DIR", "")
	t.Setenv("WORKBENCH_RUNNER_ROOT", root)
	got, err := privateUpdateSourceDir()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(root, "workbench"))
	if got != want {
		t.Fatalf("source dir=%q want %q", got, want)
	}
}

func TestDecodePrivateControlAllowsOnlyArgumentFreeUpdateAction(t *testing.T) {
	raw := []byte(`{"version":1,"id":"update-12345678","action":"update_workbench","args":{}}`)
	env, err := decodePrivateControl(raw, "update-12345678")
	if err != nil {
		t.Fatal(err)
	}
	if env.Action != "update_workbench" {
		t.Fatalf("action=%q", env.Action)
	}
	if err := decodePrivateControlArgs([]byte(`{"command":"rm -rf /"}`), &struct{}{}); err == nil {
		t.Fatal("update control must reject arbitrary arguments")
	}
}
