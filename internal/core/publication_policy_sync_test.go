package core

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestSavePublicationPolicyForExecutionHostsLocalOnly(t *testing.T) {
	isolateKnowledgeConfig(t)
	repo := initPrepareTestRepo(t)

	result, err := SavePublicationPolicyForExecutionHosts(context.Background(), repo, PublicationPrepare, "ignored", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Local.Mode != PublicationPrepare || result.Local.RemoteURL != "" || result.Runner != nil {
		t.Fatalf("unexpected prepare sync result: %#v", result)
	}

	remote := filepath.Join(t.TempDir(), "review.git")
	result, err = SavePublicationPolicyForExecutionHosts(context.Background(), repo, PublicationPublish, remote, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Local.Mode != PublicationPublish || result.Local.RemoteURL != remote || result.Runner != nil {
		t.Fatalf("unexpected publish sync result: %#v", result)
	}
}

func TestSavePublicationPolicyForExecutionHostsReportsRunnerSyncFailureAfterLocalSave(t *testing.T) {
	isolateKnowledgeConfig(t)
	repo := initPrepareTestRepo(t)
	remote := filepath.Join(t.TempDir(), "review.git")

	result, err := SavePublicationPolicyForExecutionHosts(context.Background(), repo, PublicationPublish, remote, "-oProxyCommand=evil")
	if err == nil || !strings.Contains(err.Error(), "runner sync failed") {
		t.Fatalf("expected runner sync failure, result=%#v err=%v", result, err)
	}
	if result.Local.Mode != PublicationPublish || result.Local.RemoteURL != remote {
		t.Fatalf("local policy was not preserved after runner sync failure: %#v", result)
	}
	stored, ok, readErr := PublicationPolicyFor(repo)
	if readErr != nil || !ok || stored.Mode != PublicationPublish || stored.RemoteURL != remote {
		t.Fatalf("stored local policy mismatch: %#v ok=%t err=%v", stored, ok, readErr)
	}
}
