package core

import (
	"context"
	"path/filepath"
	"testing"
)

func TestInspectChangesetCurrentCheckout(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	got, err := InspectChangeset(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Safe || got.BaseRevision == "" {
		t.Fatalf("unexpected inspection: %#v", got)
	}
}

func TestInspectChangesetRejectsRepositorySubdirectory(t *testing.T) {
	if _, err := InspectChangeset(context.Background(), "."); err == nil {
		t.Fatal("expected repository subdirectory to be rejected")
	}
}
