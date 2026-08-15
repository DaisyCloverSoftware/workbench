package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunResultReviewSurvivesRunnerJSONRoundTrip(t *testing.T) {
	original := RunnerResponse{
		Result: RunResult{
			Output: "done",
			Review: &TaskReviewResult{
				Changed:           true,
				BaseRevision:      "base123",
				Fingerprint:       "fingerprint123",
				Branch:            "workbench/task-123",
				Commit:            "abcdef0123456789",
				Files:             []string{"a.go", "b.go"},
				PublicationStatus: ReviewPublicationPublished,
				Published:         true,
			},
		},
		ProviderID:   "claude",
		ProviderName: "Claude",
	}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RunnerResponse
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Result.Review == nil {
		t.Fatal("structured review disappeared from runner response")
	}
	if decoded.Result.Review.Branch != original.Result.Review.Branch || decoded.Result.Review.Commit != original.Result.Review.Commit {
		t.Fatalf("review identity changed across runner JSON: %#v", decoded.Result.Review)
	}
	if len(decoded.Result.Review.Files) != 2 || decoded.Result.Review.PublicationStatus != ReviewPublicationPublished {
		t.Fatalf("review provenance changed across runner JSON: %#v", decoded.Result.Review)
	}
}

func TestCloneStateDeepCopiesStructuredReview(t *testing.T) {
	original := State{Tasks: []Task{{
		ID: "task-1",
		Review: &TaskReviewResult{
			Changed: true,
			Files:   []string{"one.go", "two.go"},
		},
	}}}
	cloned := cloneState(original)
	cloned.Tasks[0].Review.Files[0] = "mutated.go"
	cloned.Tasks[0].Review.Changed = false
	if original.Tasks[0].Review.Files[0] != "one.go" || !original.Tasks[0].Review.Changed {
		t.Fatalf("cloneState leaked structured review mutation into engine state: %#v", original.Tasks[0].Review)
	}
}

func TestStructuredTaskReviewNeverCarriesPublicationTarget(t *testing.T) {
	task := Task{
		ID:     "task-privacy",
		Status: TaskCompleted,
		Review: &TaskReviewResult{
			Changed:           true,
			Branch:            "workbench/task-privacy",
			Commit:            "abcdef0123456789",
			PublicationStatus: ReviewPublicationPublished,
			Published:         true,
		},
	}
	b, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"remote_url", "publication_target", "github.com/", "git@"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("structured task result exposed private publication target material %q: %s", forbidden, text)
		}
	}
}
