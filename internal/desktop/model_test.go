package desktop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestBuildSnapshotUsesProjectRegistryAndPrioritisesAttention(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	p1Path := t.TempDir()
	p2Path := t.TempDir()
	p1, err := eng.SelectProject(p1Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SaveNotes(p1.Path, "project one context"); err != nil {
		t.Fatal(err)
	}
	p2, err := eng.SelectProject(p2Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPinned(p2.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.SelectProject(p1.Path); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	st := eng.State()
	st.Tasks = []core.Task{
		{ID: "done", ProjectPath: p1.Path, Title: "Finished work", Intent: "finish it", Status: core.TaskCompleted, UpdatedAt: now},
		{ID: "needs-you", ProjectPath: p1.Path, Title: "Decision", Intent: "choose safely", Status: core.TaskNeedsAttention, AttentionQuestion: "Choose A or B", UpdatedAt: now.Add(-time.Minute)},
		{ID: "working", ProjectPath: p2.Path, Title: "Other project", Intent: "work elsewhere", Status: core.TaskRunning, UpdatedAt: now},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err = core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := BuildSnapshot(eng, "")
	if snapshot.ActiveProjectID != p1.ID || snapshot.ActivePath != p1.Path || snapshot.ActiveNotes != "project one context" {
		t.Fatalf("active project snapshot wrong: %#v", snapshot)
	}
	if len(snapshot.Projects) != 2 || snapshot.Projects[0].ID != p2.ID || !snapshot.Projects[0].Pinned {
		t.Fatalf("project ordering/metadata wrong: %#v", snapshot.Projects)
	}
	if len(snapshot.Tasks) != 2 {
		t.Fatalf("active project tasks=%d want 2", len(snapshot.Tasks))
	}
	if snapshot.Summary.NeedsHuman != 1 || snapshot.Summary.Completed != 1 || snapshot.Summary.Active != 0 {
		t.Fatalf("active project summary=%#v", snapshot.Summary)
	}
	if snapshot.SelectedTaskID != "needs-you" {
		t.Fatalf("selected task=%q want needs-you", snapshot.SelectedTaskID)
	}
	selected, ok := snapshot.SelectedTask()
	if !ok || !selected.NeedsHuman || selected.AttentionQuestion != "Choose A or B" {
		t.Fatalf("selected task card=%#v ok=%t", selected, ok)
	}
}

func TestBuildSnapshotPreservesValidSelectionBeforeAttentionPriority(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	st := eng.State()
	st.Tasks = []core.Task{
		{ID: "requested", ProjectPath: project.Path, Title: "Requested", Status: core.TaskCompleted},
		{ID: "attention", ProjectPath: project.Path, Title: "Attention", Status: core.TaskNeedsAttention, AttentionQuestion: "Need input"},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err = core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := BuildSnapshot(eng, "requested")
	if snapshot.SelectedTaskID != "requested" {
		t.Fatalf("valid explicit selection was replaced: %q", snapshot.SelectedTaskID)
	}
}

func TestFirstAttentionTargetUsesSidebarProjectOrder(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	unpinned, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	pinned, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.SetProjectPinned(pinned.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.SelectProject(unpinned.Path); err != nil {
		t.Fatal(err)
	}

	st := eng.State()
	st.Tasks = []core.Task{
		{ID: "unpinned-attention", ProjectPath: unpinned.Path, Status: core.TaskNeedsAttention},
		{ID: "pinned-done", ProjectPath: pinned.Path, Status: core.TaskCompleted},
		{ID: "pinned-attention", ProjectPath: pinned.Path, Status: core.TaskNeedsAttention},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err = core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}

	target, ok := FirstAttentionTarget(eng)
	if !ok {
		t.Fatal("expected a global human-attention target")
	}
	if target.ProjectID != pinned.ID || target.TaskID != "pinned-attention" {
		t.Fatalf("attention target=%#v want pinned project/task", target)
	}
	active, ok := eng.ActiveProject()
	if !ok || active.ID != unpinned.ID {
		t.Fatalf("read-only attention lookup changed active project: %#v ok=%t", active, ok)
	}
}

func TestFirstAttentionTargetReturnsFalseWithoutHumanBoundary(t *testing.T) {
	store, err := core.NewStoreAt(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	project, err := eng.SelectProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	st := eng.State()
	st.Tasks = []core.Task{{ID: "working", ProjectPath: project.Path, Status: core.TaskRunning}}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	eng, err = core.NewEngine(store)
	if err != nil {
		t.Fatal(err)
	}
	if target, ok := FirstAttentionTarget(eng); ok {
		t.Fatalf("unexpected attention target: %#v", target)
	}
}

func TestTaskItemCarriesStructuredReviewAndPullRequestFacts(t *testing.T) {
	task := core.Task{
		ID:          "review",
		Title:       "Review me",
		Status:      core.TaskCompleted,
		ProjectPath: t.TempDir(),
		Review: &core.TaskReviewResult{
			Changed:           true,
			Branch:            "workbench/review",
			Commit:            "0123456789abcdef",
			Files:             []string{"one.go", "two.go"},
			PublicationStatus: core.ReviewPublicationPublished,
			Published:         true,
			PullRequestStatus: core.ReviewPullRequestAvailable,
			PullRequestNumber: 42,
			PullRequestState:  "open",
		},
	}
	item := taskItems([]core.Task{task})[0]
	if item.ReviewBranch != "workbench/review" || item.ReviewFiles != 2 || item.PullRequestNumber != 42 || item.PullRequestStatus != core.ReviewPullRequestAvailable {
		t.Fatalf("structured review lost in desktop task card: %#v", item)
	}
}
