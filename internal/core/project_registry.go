package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const projectStateVersion = 3

func normalizeProjectRegistryState(st State) State {
	now := time.Now().UTC()
	migrating := st.Version < projectStateVersion
	projects := make([]Project, 0, len(st.Projects)+1)
	index := map[string]int{}

	add := func(path, name, notes string, pinned bool, addedAt, lastUsedAt time.Time) string {
		path = normalizeProjectPath(path)
		if path == "" {
			return ""
		}
		id := projectID(path)
		if id == "" {
			return ""
		}
		if i, ok := index[id]; ok {
			p := &projects[i]
			if strings.TrimSpace(name) != "" {
				p.Name = cleanProjectName(name, path)
			}
			if notes != "" {
				p.Notes = notes
			}
			p.Pinned = p.Pinned || pinned
			if !addedAt.IsZero() && (p.AddedAt.IsZero() || addedAt.Before(p.AddedAt)) {
				p.AddedAt = addedAt.UTC()
			}
			if lastUsedAt.After(p.LastUsedAt) {
				p.LastUsedAt = lastUsedAt.UTC()
			}
			return id
		}
		if addedAt.IsZero() {
			addedAt = now
		}
		if lastUsedAt.IsZero() {
			lastUsedAt = addedAt
		}
		index[id] = len(projects)
		projects = append(projects, Project{
			ID:         id,
			Path:       path,
			Name:       cleanProjectName(name, path),
			Notes:      notes,
			Pinned:     pinned,
			AddedAt:    addedAt.UTC(),
			LastUsedAt: lastUsedAt.UTC(),
		})
		return id
	}

	for _, p := range st.Projects {
		add(p.Path, p.Name, p.Notes, p.Pinned, p.AddedAt, p.LastUsedAt)
	}

	legacyActivePath := normalizeProjectPath(st.ProjectPath)
	legacyActiveID := ""
	if legacyActivePath != "" {
		legacyActiveID = add(legacyActivePath, "", st.Notes, false, now, now)
	}
	// Historic Task.ProjectPath is the only multi-project information available
	// in v0.7-era state. Import it once while moving to v3. After that the
	// explicit registry is authoritative: removing a project must not cause old
	// task history to resurrect it on the next save.
	if migrating {
		for _, task := range st.Tasks {
			used := task.UpdatedAt
			if used.IsZero() {
				used = task.CreatedAt
			}
			add(task.ProjectPath, "", "", false, task.CreatedAt, used)
		}
	}

	activeID := strings.TrimSpace(st.ActiveProjectID)
	if _, ok := index[activeID]; !ok {
		activeID = legacyActiveID
	}
	if activeID == "" && len(projects) > 0 {
		activeID = mostRecentProjectID(projects)
	}

	st.Version = projectStateVersion
	st.Projects = projects
	st.ActiveProjectID = activeID
	mirrorActiveProject(&st)
	return st
}

func normalizeProjectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func projectPathIdentity(path string) string {
	path = normalizeProjectPath(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func projectID(path string) string {
	identity := projectPathIdentity(path)
	if identity == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(identity))
	return "project-" + hex.EncodeToString(sum[:8])
}

func cleanProjectName(name, path string) string {
	name = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "\r", " "), "\n", " "))
	if len(name) > 100 {
		name = strings.TrimSpace(name[:100])
	}
	if name != "" {
		return name
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "." || base == string(filepath.Separator) || strings.TrimSpace(base) == "" {
		return "Project"
	}
	return base
}

func sameProjectPath(a, b string) bool {
	a = projectPathIdentity(a)
	b = projectPathIdentity(b)
	return a != "" && a == b
}

func mostRecentProjectID(projects []Project) string {
	if len(projects) == 0 {
		return ""
	}
	best := projects[0]
	for _, p := range projects[1:] {
		if p.LastUsedAt.After(best.LastUsedAt) {
			best = p
		}
	}
	return best.ID
}

func mirrorActiveProject(st *State) {
	if st == nil {
		return
	}
	st.ProjectPath = ""
	st.Notes = ""
	for _, p := range st.Projects {
		if p.ID == st.ActiveProjectID {
			st.ProjectPath = p.Path
			st.Notes = p.Notes
			return
		}
	}
	if len(st.Projects) == 0 {
		st.ActiveProjectID = ""
	}
}

func upsertProjectLocked(st *State, path string, touch bool) (Project, error) {
	if st == nil {
		return Project{}, errors.New("state is nil")
	}
	path = normalizeProjectPath(path)
	if path == "" {
		return Project{}, errors.New("project path is empty")
	}
	id := projectID(path)
	now := time.Now().UTC()
	for i := range st.Projects {
		if st.Projects[i].ID == id || sameProjectPath(st.Projects[i].Path, path) {
			st.Projects[i].Path = path
			if strings.TrimSpace(st.Projects[i].Name) == "" {
				st.Projects[i].Name = cleanProjectName("", path)
			}
			if st.Projects[i].AddedAt.IsZero() {
				st.Projects[i].AddedAt = now
			}
			if touch || st.Projects[i].LastUsedAt.IsZero() {
				st.Projects[i].LastUsedAt = now
			}
			return st.Projects[i], nil
		}
	}
	project := Project{
		ID:         id,
		Path:       path,
		Name:       cleanProjectName("", path),
		AddedAt:    now,
		LastUsedAt: now,
	}
	st.Projects = append(st.Projects, project)
	return project, nil
}

func canonicalProjectSelection(path string) (string, error) {
	path = normalizeProjectPath(path)
	if path == "" {
		return "", errors.New("project path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("project path is not a directory")
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		if abs, absErr := filepath.Abs(resolved); absErr == nil {
			path = filepath.Clean(abs)
		}
	}
	return path, nil
}

func sortedProjects(projects []Project) []Project {
	out := append([]Project(nil), projects...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		if !out[i].LastUsedAt.Equal(out[j].LastUsedAt) {
			return out[i].LastUsedAt.After(out[j].LastUsedAt)
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
