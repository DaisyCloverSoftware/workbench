package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineKnowledgeStoreFollowsStateDirectory(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	knowledge, err := eng.KnowledgeStore()
	if err != nil {
		t.Fatal(err)
	}
	if knowledge.Path() != filepath.Join(dir, "knowledge.json") {
		t.Fatalf("knowledge path = %q", knowledge.Path())
	}
}

func TestIntentWithKnowledgeAttachesLatestCheckpoint(t *testing.T) {
	store, err := NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.SaveCheckpoint(project, "Keep the durable checkpoint attached to future autonomous work.", []string{"Do not replay entire chat transcripts."}, nil, []string{"Continue from compact context."}); err != nil {
		t.Fatal(err)
	}
	intent := eng.IntentWithKnowledge("Implement the next memory task.", project)
	for _, want := range []string{"Implement the next memory task.", "WORKBENCH COMPACT CONTEXT", "Do not replay entire chat transcripts."} {
		if !strings.Contains(intent, want) {
			t.Fatalf("worker intent missing %q:\n%s", want, intent)
		}
	}
}
