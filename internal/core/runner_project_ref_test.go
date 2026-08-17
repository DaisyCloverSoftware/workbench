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
	rootNumber, name, ok := RunnerProjectLocator(ref)
	if !ok || rootNumber != 0 || name != "garage" {
		t.Fatalf("unexpected locator root=%d name=%q ok=%v", rootNumber, name, ok)
	}
	if got, ok := RunnerProjectName(ref); !ok || got != "garage" {
		t.Fatalf("unexpected round trip %q %v", got, ok)
	}
	if !IsRunnerProjectReference(ref) {
		t.Fatal("expected runner project reference")
	}
}

func TestRunnerScopedProjectReferenceRoundTrip(t *testing.T) {
	ref, err := RunnerScopedProjectReference(2, "garage")
	if err != nil {
		t.Fatal(err)
	}
	if ref != "runner://r2/garage" {
		t.Fatalf("unexpected scoped ref %q", ref)
	}
	rootNumber, name, ok := RunnerProjectLocator(ref)
	if !ok || rootNumber != 2 || name != "garage" {
		t.Fatalf("unexpected locator root=%d name=%q ok=%v", rootNumber, name, ok)
	}
	normalized, ok := NormalizeRunnerProjectReference("RUNNER://R2/garage")
	if !ok || normalized != "runner://r2/garage" {
		t.Fatalf("unexpected normalized scoped ref %q %v", normalized, ok)
	}
}

func TestRunnerProjectReferenceRejectsTraversal(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../repo", "a/b", `a\\b`, "a:b"} {
		if _, err := RunnerProjectReference(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
	for _, ref := range []string{"runner://r0/repo", "runner://r-1/repo", "runner://r1/../repo", "runner://r1/a/b", "runner://other/repo"} {
		if IsRunnerProjectReference(ref) {
			t.Fatalf("expected scoped ref %q to be rejected", ref)
		}
	}
}
