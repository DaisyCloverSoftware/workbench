package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKnownProjectPublicationPolicyReadDoesNotRequireLiveGit(t *testing.T) {
	isolateKnowledgeConfig(t)
	repo := initPrepareTestRepo(t)
	saved, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPrepare})
	if err != nil {
		t.Fatal(err)
	}

	// The read path is used by the Windows Settings UI. Once Workbench has a
	// canonical registered project, merely displaying a saved policy must not
	// execute Git or depend on the repository still being responsive.
	if err := os.RemoveAll(filepath.Join(repo, ".git")); err != nil {
		t.Fatal(err)
	}
	got, ok, err := PublicationPolicyForKnownProject(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.Project != saved.Project || got.Mode != saved.Mode {
		t.Fatalf("known-project policy lookup=%#v ok=%t want %#v", got, ok, saved)
	}
}
