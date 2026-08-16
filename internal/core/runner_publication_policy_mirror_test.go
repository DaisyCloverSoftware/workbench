package core

import "testing"

func TestPublicationPolicyReadKeyPreservesRunnerReference(t *testing.T) {
	key, err := publicationPolicyReadKey("runner://garage")
	if err != nil {
		t.Fatal(err)
	}
	if key != "runner://garage" {
		t.Fatalf("unexpected runner policy key %q", key)
	}
}

func TestRunnerPublicationPolicyMirrorRequiresRunnerReference(t *testing.T) {
	if _, err := SaveRunnerProjectPublicationPolicyMirror(PublicationPolicy{Project: "not-runner", Mode: PublicationPrepare}); err == nil {
		t.Fatal("expected local path to be rejected by runner mirror")
	}
}
