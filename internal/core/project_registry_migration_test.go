package core

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyProjectRegistryMigrationBoundary(t *testing.T) {
	active := filepath.Join(t.TempDir(), "active-project")
	historic := filepath.Join(t.TempDir(), "historic-project")
	legacy := State{
		Version:     2,
		ProjectPath: active,
		Notes:       "legacy active notes",
		Tasks: []Task{{
			ID:          "historic-task",
			ProjectPath: historic,
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-time.Hour),
		}},
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	var decoded State
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ProjectPath != active {
		t.Fatalf("legacy JSON lost project_path before migration: got=%q want=%q json=%s", decoded.ProjectPath, active, string(b))
	}
	if decoded.Version != 2 || len(decoded.Tasks) != 1 || decoded.Tasks[0].ProjectPath != historic {
		t.Fatalf("legacy JSON decoded unexpectedly: version=%d tasks=%#v json=%s", decoded.Version, decoded.Tasks, string(b))
	}
	migrated := normalizeProjectRegistryState(decoded)
	if len(migrated.Projects) != 2 {
		t.Fatalf("direct migration lost a project: active=%q historic=%q activeID=%q legacyMirror=%q projects=%#v", active, historic, migrated.ActiveProjectID, migrated.ProjectPath, migrated.Projects)
	}
}
