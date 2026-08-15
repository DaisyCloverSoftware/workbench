package core

import "testing"

func TestTouchProjectStateDoesNotStealExistingActiveProject(t *testing.T) {
	first := t.TempDir()
	background := t.TempDir()
	st := DefaultState()

	touchProjectState(&st, first)
	firstID := projectID(first)
	if st.ActiveProjectID != firstID || !sameProjectPath(st.ProjectPath, first) {
		t.Fatalf("first project did not become active: %#v", st)
	}

	touchProjectState(&st, background)
	if st.ActiveProjectID != firstID || !sameProjectPath(st.ProjectPath, first) {
		t.Fatalf("background project use stole active workspace: %#v", st)
	}
	if len(st.Projects) != 2 {
		t.Fatalf("background project was not registered: %#v", st.Projects)
	}
	found := false
	for _, project := range st.Projects {
		if sameProjectPath(project.Path, background) {
			found = true
			if project.LastUsedAt.IsZero() {
				t.Fatal("background project use did not update recency")
			}
		}
	}
	if !found {
		t.Fatal("background project missing from registry")
	}
}

func TestTouchProjectStateRepairsMissingActiveSelection(t *testing.T) {
	project := t.TempDir()
	st := DefaultState()
	st.ActiveProjectID = "project-missing"

	touchProjectState(&st, project)
	if st.ActiveProjectID != projectID(project) || !sameProjectPath(st.ProjectPath, project) {
		t.Fatalf("touch did not repair missing active project: %#v", st)
	}
}
