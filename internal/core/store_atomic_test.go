package core

import (
	"path/filepath"
	"testing"
)

func TestStoreSaveLeavesReadableSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := NewStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	state := DefaultState()
	state.Notes = "saved snapshot"
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Notes != state.Notes {
		t.Fatalf("notes=%q want %q", loaded.Notes, state.Notes)
	}
}
