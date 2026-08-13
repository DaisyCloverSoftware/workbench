package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type IdeaStatus string

const (
	IdeaParked    IdeaStatus = "parked"
	IdeaActive    IdeaStatus = "active"
	IdeaDone      IdeaStatus = "done"
	IdeaDismissed IdeaStatus = "dismissed"
)

type Idea struct {
	ID        string         `json:"id"`
	Scope     KnowledgeScope `json:"scope"`
	Project   string         `json:"project,omitempty"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	Tags      []string       `json:"tags,omitempty"`
	Status    IdeaStatus     `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ideaState struct {
	Version int    `json:"version"`
	Ideas   []Idea `json:"ideas"`
}

const maxIdeaStateBytes = 4 << 20

var ideaMu sync.RWMutex

func IdeaStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "ideas.json"), nil
}

func SaveIdea(idea Idea) (Idea, error) {
	idea.ID = strings.TrimSpace(idea.ID)
	idea.Project = strings.TrimSpace(idea.Project)
	idea.Title = strings.TrimSpace(idea.Title)
	idea.Content = strings.TrimSpace(idea.Content)
	idea.Tags = normalizeTags(idea.Tags)
	if idea.Title == "" || idea.Content == "" {
		return Idea{}, errors.New("idea title and content are required")
	}
	if idea.Scope != ScopeGlobal && idea.Scope != ScopeProject {
		return Idea{}, errors.New("idea scope must be global or project")
	}
	if idea.Scope == ScopeProject && idea.Project == "" {
		return Idea{}, errors.New("project-scoped idea requires a project")
	}
	if idea.Scope == ScopeGlobal {
		idea.Project = ""
	}
	if idea.Status == "" {
		idea.Status = IdeaParked
	}
	if !validIdeaStatus(idea.Status) {
		return Idea{}, errors.New("idea status must be parked, active, done or dismissed")
	}
	if LooksSecret(idea.Title + "\n" + idea.Content + "\n" + strings.Join(idea.Tags, "\n")) {
		return Idea{}, errors.New("idea appears to contain secret material; store secrets in the vault")
	}

	release, err := lockIdeaWrite()
	if err != nil {
		return Idea{}, err
	}
	defer release()
	st, err := loadIdeaState()
	if err != nil {
		return Idea{}, err
	}
	now := time.Now().UTC()
	if idea.ID != "" {
		for i := range st.Ideas {
			if st.Ideas[i].ID != idea.ID {
				continue
			}
			old := st.Ideas[i]
			// IDs are not capabilities for moving private context between projects.
			if old.Scope != idea.Scope || old.Project != idea.Project {
				return Idea{}, errors.New("idea scope/project cannot be changed by update")
			}
			idea.CreatedAt = old.CreatedAt
			idea.UpdatedAt = now
			st.Ideas[i] = idea
			if err := saveIdeaState(st); err != nil {
				return Idea{}, err
			}
			return idea, nil
		}
		return Idea{}, errors.New("idea not found")
	}
	idea.ID = newID("idea")
	idea.CreatedAt = now
	idea.UpdatedAt = now
	st.Ideas = append(st.Ideas, idea)
	if err := saveIdeaState(st); err != nil {
		return Idea{}, err
	}
	return idea, nil
}

func ListIdeas(project string, status IdeaStatus, limit int) ([]Idea, error) {
	return SearchIdeas(project, "", status, limit)
}

func SearchIdeas(project, query string, status IdeaStatus, limit int) ([]Idea, error) {
	ideaMu.RLock()
	defer ideaMu.RUnlock()
	st, err := loadIdeaState()
	if err != nil {
		return nil, err
	}
	project = strings.TrimSpace(project)
	if status != "" && !validIdeaStatus(status) {
		return nil, errors.New("unknown idea status")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	terms := searchTerms(query)
	type scoredIdea struct {
		idea  Idea
		score int
	}
	found := make([]scoredIdea, 0)
	for _, idea := range st.Ideas {
		if idea.Scope == ScopeProject && idea.Project != project {
			continue
		}
		if status != "" && idea.Status != status {
			continue
		}
		score := 1
		if len(terms) > 0 {
			score = 0
			hay := strings.ToLower(idea.Title + "\n" + idea.Content + "\n" + strings.Join(idea.Tags, " "))
			for _, term := range terms {
				if strings.Contains(hay, term) {
					score += 3
				}
				if strings.Contains(strings.ToLower(idea.Title), term) {
					score += 3
				}
			}
		}
		if idea.Scope == ScopeProject && idea.Project == project && project != "" {
			score++
		}
		if score > 0 {
			found = append(found, scoredIdea{idea: idea, score: score})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].idea.UpdatedAt.After(found[j].idea.UpdatedAt)
	})
	out := make([]Idea, 0, min(limit, len(found)))
	for i := 0; i < len(found) && i < limit; i++ {
		out = append(out, found[i].idea)
	}
	return out, nil
}

func SetIdeaStatus(project, id string, status IdeaStatus) (Idea, error) {
	project = strings.TrimSpace(project)
	id = strings.TrimSpace(id)
	if id == "" || !validIdeaStatus(status) {
		return Idea{}, errors.New("valid idea id and status are required")
	}
	release, err := lockIdeaWrite()
	if err != nil {
		return Idea{}, err
	}
	defer release()
	st, err := loadIdeaState()
	if err != nil {
		return Idea{}, err
	}
	for i := range st.Ideas {
		idea := &st.Ideas[i]
		if idea.ID != id {
			continue
		}
		if idea.Scope == ScopeProject && idea.Project != project {
			return Idea{}, errors.New("idea not found in this project")
		}
		idea.Status = status
		idea.UpdatedAt = time.Now().UTC()
		result := *idea
		result.Tags = append([]string(nil), idea.Tags...)
		if err := saveIdeaState(st); err != nil {
			return Idea{}, err
		}
		return result, nil
	}
	return Idea{}, errors.New("idea not found")
}

func validIdeaStatus(status IdeaStatus) bool {
	switch status {
	case IdeaParked, IdeaActive, IdeaDone, IdeaDismissed:
		return true
	default:
		return false
	}
}

func loadIdeaState() (ideaState, error) {
	path, err := IdeaStatePath()
	if err != nil {
		return ideaState{}, err
	}
	info, statErr := os.Stat(path)
	if errors.Is(statErr, os.ErrNotExist) {
		return ideaState{Version: 1}, nil
	}
	if statErr != nil {
		return ideaState{}, statErr
	}
	if info.Size() > maxIdeaStateBytes {
		return ideaState{}, errors.New("idea store is unexpectedly large")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ideaState{}, err
	}
	var st ideaState
	if err := json.Unmarshal(b, &st); err != nil {
		return ideaState{}, err
	}
	if st.Version == 0 {
		st.Version = 1
	}
	return st, nil
}

func saveIdeaState(st ideaState) error {
	path, err := IdeaStatePath()
	if err != nil {
		return err
	}
	st.Version = 1
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > maxIdeaStateBytes {
		return errors.New("idea store exceeds its local size limit")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ideas-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func lockIdeaWrite() (func(), error) {
	ideaMu.Lock()
	path, err := IdeaStatePath()
	if err != nil {
		ideaMu.Unlock()
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, openErr := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() {
				_ = os.Remove(lockPath)
				ideaMu.Unlock()
			}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			ideaMu.Unlock()
			return nil, openErr
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 2*time.Minute {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			ideaMu.Unlock()
			return nil, errors.New("timed out waiting for Workbench idea-store lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
