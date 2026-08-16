package core

import "testing"

func TestRunnerProjectReferenceRoundTrip(t *testing.T) {
	ref, err := RunnerProjectReference("garage")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "runner://garage" {
		t.Fatalf("unexpected ref %q", ref)
	}
	name, ok := RunnerProjectName(ref)
	if !ok || name != "garage" {
		t.Fatalf("unexpected round trip %q %v", name, ok)
	}
	if !IsRunnerProjectReference(ref) {
		t.Fatal("expected runner project reference")
	}
}

func TestRunnerProjectReferenceRejectsTraversal(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../repo", "a/b", `a\\b`, "a:b"} {
		if _, err := RunnerProjectReference(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}
