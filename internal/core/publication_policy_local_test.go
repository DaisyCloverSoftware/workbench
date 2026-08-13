package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPublicationPreparePolicyRoundTrip(t *testing.T) {
	isolateKnowledgeConfig(t)
	repo := initPrepareTestRepo(t)
	saved, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: PublicationPrepare})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Project == "" || saved.Mode != PublicationPrepare || saved.RemoteURL != "" || saved.UpdatedAt.IsZero() {
		t.Fatalf("unexpected saved policy: %#v", saved)
	}
	got, ok, err := PublicationPolicyFor(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.Project != saved.Project || got.Mode != PublicationPrepare {
		t.Fatalf("unexpected loaded policy: %#v ok=%t", got, ok)
	}

	path, err := PublicationPolicyStatePath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "publication-policy.json" {
		t.Fatalf("unexpected policy path: %s", path)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("policy store permissions are too broad: %o", info.Mode().Perm())
		}
	}

	if err := DeletePublicationPolicy(repo); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := PublicationPolicyFor(repo); err != nil || ok {
		t.Fatalf("policy still present after delete: ok=%t err=%v", ok, err)
	}
}

func TestPublicationPolicyRequiresRepositoryRootAndKnownMode(t *testing.T) {
	isolateKnowledgeConfig(t)
	repo := initPrepareTestRepo(t)
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := SavePublicationPolicy(PublicationPolicy{Project: sub, Mode: PublicationPrepare}); err == nil {
		t.Fatal("expected subdirectory policy to be rejected")
	}
	if _, err := SavePublicationPolicy(PublicationPolicy{Project: repo, Mode: "unknown"}); err == nil {
		t.Fatal("expected unknown publication mode to be rejected")
	}
}
