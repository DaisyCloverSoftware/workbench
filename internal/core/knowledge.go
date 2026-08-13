package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type MemoryScope string

type MemoryKind string

const (
	MemoryProject MemoryScope = "project"
	MemoryGlobal  MemoryScope = "global"

	MemoryDecision   MemoryKind = "decision"
	MemoryConstraint MemoryKind = "constraint"
	MemoryLesson     MemoryKind = "lesson"
	MemoryPattern    MemoryKind = "pattern"
	MemoryOutcome    MemoryKind = "outcome"
)

type ProjectIdentity struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Remote  string `json:"remote,omitempty"`
	Portable bool  `json:"portable"`
}

type MemoryItem struct {
	ID           string       `json:"id"`
	Scope        MemoryScope  `json:"scope"`
	ProjectKey   string       `json:"project_key,omitempty"`
	ProjectName  string       `json:"project_name,omitempty"`
	Kind         MemoryKind   `json:"kind"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary"`
	Content      string       `json:"content,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Source       string       `json:"source,omitempty"`
	SourceTaskID string       `json:"source_task_id,omitempty"`
	Fingerprint  string       `json:"fingerprint"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type ContextCheckpoint struct {
	ID          string    `json:"id"`
	ProjectKey  string    `json:"project_key"`
	ProjectName string    `json:"project_name"`
	Summary     string    `json:"summary"`
	Decisions   []string  `json:"decisions,omitempty"`
	OpenLoops   []string  `json:"open_loops,omitempty"`
	NextActions []string  `json:"next_actions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Routine struct {
	ID          string      `json:"id"`
	Scope       MemoryScope `json:"scope"`
	ProjectKey  string      `json:"project_key,omitempty"`
	ProjectName string      `json:"project_name,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Triggers    []string    `json:"triggers,omitempty"`
	Steps       []string    `json:"steps,omitempty"`
	Code        string      `json:"code,omitempty"`
	Language    string      `json:"language,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type ContextPack struct {
	Version        int                `json:"version"`
	Project        ProjectIdentity    `json:"project"`
	Query          string             `json:"query,omitempty"`
	Checkpoint     *ContextCheckpoint `json:"checkpoint,omitempty"`
	Memories       []MemoryItem       `json:"memories,omitempty"`
	Routines       []Routine          `json:"routines,omitempty"`
	ContextText    string             `json:"context_text"`
	EstimatedChars int                `json:"estimated_chars"`
	GeneratedAt    time.Time          `json:"generated_at"`
}

type knowledgeState struct {
	Version     int                 `json:"version"`
	Memories    []MemoryItem        `json:"memories"`
	Checkpoints []ContextCheckpoint `json:"checkpoints"`
	Routines    []Routine           `json:"routines"`
}

type KnowledgeStore struct {
	mu   sync.Mutex
	path string
}

func NewKnowledgeStoreAt(path string) (*KnowledgeStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("knowledge path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return &KnowledgeStore{path: path}, nil
}

func NewKnowledgeStore() (*KnowledgeStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewKnowledgeStoreAt(filepath.Join(dir, "Workbench", "knowledge.json"))
}

func (s *KnowledgeStore) Path() string { return s.path }

func (s *KnowledgeStore) loadLocked() (knowledgeState, error) {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return knowledgeState{Version: 1}, nil
	}
	if err != nil {
		return knowledgeState{}, err
	}
	var st knowledgeState
	if err := json.Unmarshal(b, &st); err != nil {
		return knowledgeState{}, err
	}
	if st.Version == 0 {
		st.Version = 1
	}
	return st, nil
}

func (s *KnowledgeStore) saveLocked(st knowledgeState) error {
	st.Version = 1
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func IdentifyProject(project string) (ProjectIdentity, error) {
	root, err := projectRoot(project)
	if err != nil {
		return ProjectIdentity{}, err
	}
	identity := ProjectIdentity{Path: root, Name: filepath.Base(root)}
	cmd := exec.Command("git", "-C", root, "remote", "get-url", "origin")
	if out, cmdErr := cmd.Output(); cmdErr == nil {
		if remote := normalizeGitRemote(strings.TrimSpace(string(out))); remote != "" {
			identity.Remote = remote
			identity.Key = "git:" + strings.ToLower(remote)
			identity.Portable = true
			return identity, nil
		}
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(root))))
	identity.Key = "local:" + strings.ToLower(identity.Name) + ":" + hex.EncodeToString(sum[:6])
	return identity, nil
}

func normalizeGitRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(raw, "git@"), ":", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[0]+"/"+strings.TrimPrefix(parts[1], "/"), ".git")
		}
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		host := strings.TrimPrefix(strings.TrimSpace(u.Host), "git@")
		path := strings.TrimPrefix(strings.TrimSpace(u.Path), "/")
		path = strings.TrimSuffix(path, ".git")
		if host != "" && path != "" {
			return host + "/" + path
		}
	}
	return strings.TrimSuffix(raw, ".git")
}

func validMemoryScope(scope MemoryScope) bool {
	return scope == MemoryProject || scope == MemoryGlobal
}

func validMemoryKind(kind MemoryKind) bool {
	switch kind {
	case MemoryDecision, MemoryConstraint, MemoryLesson, MemoryPattern, MemoryOutcome:
		return true
	default:
		return false
	}
}

func (s *KnowledgeStore) Remember(project string, scope MemoryScope, kind MemoryKind, title, summary, content string, tags []string, source string) (MemoryItem, error) {
	return s.remember(project, scope, kind, title, summary, content, tags, source, "")
}

func (s *KnowledgeStore) remember(project string, scope MemoryScope, kind MemoryKind, title, summary, content string, tags []string, source, sourceTaskID string) (MemoryItem, error) {
	if !validMemoryScope(scope) {
		return MemoryItem{}, errors.New("memory scope must be project or global")
	}
	if !validMemoryKind(kind) {
		return MemoryItem{}, errors.New("unsupported memory kind")
	}
	title = cleanText(title, 200)
	summary = cleanText(summary, 4000)
	content = cleanText(content, 32000)
	if title == "" || summary == "" {
		return MemoryItem{}, errors.New("memory title and summary are required")
	}
	tags = cleanList(tags, 32, 64)
	if LooksSecret(strings.Join(append([]string{title, summary, content}, tags...), "\n")) {
		return MemoryItem{}, errors.New("memory appears to contain secret material; store secrets in the vault instead")
	}

	var projectID ProjectIdentity
	var err error
	if scope == MemoryProject {
		projectID, err = IdentifyProject(project)
		if err != nil {
			return MemoryItem{}, err
		}
	}
	now := time.Now().UTC()
	fingerprint := memoryFingerprint(scope, projectID.Key, kind, title, summary, content, tags)
	item := MemoryItem{
		ID:           newID("mem"),
		Scope:        scope,
		ProjectKey:   projectID.Key,
		ProjectName:  projectID.Name,
		Kind:         kind,
		Title:        title,
		Summary:      summary,
		Content:      content,
		Tags:         tags,
		Source:       cleanText(source, 200),
		SourceTaskID: cleanText(sourceTaskID, 120),
		Fingerprint:  fingerprint,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadLocked()
	if err != nil {
		return MemoryItem{}, err
	}
	for i := range st.Memories {
		if st.Memories[i].Fingerprint == fingerprint {
			item.ID = st.Memories[i].ID
			item.CreatedAt = st.Memories[i].CreatedAt
			st.Memories[i] = item
			if err := s.saveLocked(st); err != nil {
				return MemoryItem{}, err
			}
			return item, nil
		}
	}
	st.Memories = append(st.Memories, item)
	if len(st.Memories) > 5000 {
		sort.Slice(st.Memories, func(i, j int) bool { return st.Memories[i].UpdatedAt.After(st.Memories[j].UpdatedAt) })
		st.Memories = st.Memories[:5000]
	}
	if err := s.saveLocked(st); err != nil {
		return MemoryItem{}, err
	}
	return item, nil
}

func (s *KnowledgeStore) Recall(project, query string, limit int) ([]MemoryItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	var projectID ProjectIdentity
	var err error
	if strings.TrimSpace(project) != "" {
		projectID, err = IdentifyProject(project)
		if err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return rankMemories(st.Memories, projectID.Key, query, limit), nil
}

func rankMemories(items []MemoryItem, projectKey, query string, limit int) []MemoryItem {
	tokens := queryTokens(query)
	type scored struct {
		item  MemoryItem
		score int
	}
	var ranked []scored
	for _, item := range items {
		if item.Scope == MemoryProject && (projectKey == "" || item.ProjectKey != projectKey) {
			continue
		}
		score := 0
		if item.Scope == MemoryProject {
			score += 24
		} else {
			score += 8
		}
		switch item.Kind {
		case MemoryDecision, MemoryConstraint:
			score += 6
		case MemoryPattern, MemoryLesson:
			score += 4
		}
		if len(tokens) == 0 {
			score += 1
		} else {
			title := strings.ToLower(item.Title)
			summary := strings.ToLower(item.Summary)
			tags := strings.ToLower(strings.Join(item.Tags, " "))
			content := strings.ToLower(item.Content)
			matches := 0
			for _, token := range tokens {
				if strings.Contains(title, token) {
					score += 10
					matches++
				}
				if strings.Contains(tags, token) {
					score += 7
					matches++
				}
				if strings.Contains(summary, token) {
					score += 5
					matches++
				}
				if content != "" && strings.Contains(content, token) {
					score += 2
					matches++
				}
			}
			if matches == 0 {
				continue
			}
		}
		ranked = append(ranked, scored{item: item, score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].item.UpdatedAt.After(ranked[j].item.UpdatedAt)
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]MemoryItem, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].item
	}
	return out
}

func (s *KnowledgeStore) SaveCheckpoint(project, summary string, decisions, openLoops, nextActions []string) (ContextCheckpoint, error) {
	identity, err := IdentifyProject(project)
	if err != nil {
		return ContextCheckpoint{}, err
	}
	summary = cleanText(summary, 8000)
	if summary == "" {
		return ContextCheckpoint{}, errors.New("checkpoint summary is required")
	}
	decisions = cleanList(decisions, 32, 1200)
	openLoops = cleanList(openLoops, 32, 1200)
	nextActions = cleanList(nextActions, 32, 1200)
	all := append([]string{summary}, decisions...)
	all = append(all, openLoops...)
	all = append(all, nextActions...)
	if LooksSecret(strings.Join(all, "\n")) {
		return ContextCheckpoint{}, errors.New("checkpoint appears to contain secret material")
	}
	checkpoint := ContextCheckpoint{
		ID:          newID("checkpoint"),
		ProjectKey:  identity.Key,
		ProjectName: identity.Name,
		Summary:     summary,
		Decisions:   decisions,
		OpenLoops:   openLoops,
		NextActions: nextActions,
		CreatedAt:   time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadLocked()
	if err != nil {
		return ContextCheckpoint{}, err
	}
	st.Checkpoints = append(st.Checkpoints, checkpoint)
	st.Checkpoints = trimProjectCheckpoints(st.Checkpoints, identity.Key, 24)
	if err := s.saveLocked(st); err != nil {
		return ContextCheckpoint{}, err
	}
	return checkpoint, nil
}

func trimProjectCheckpoints(all []ContextCheckpoint, projectKey string, keep int) []ContextCheckpoint {
	count := 0
	out := make([]ContextCheckpoint, 0, len(all))
	for i := len(all) - 1; i >= 0; i-- {
		cp := all[i]
		if cp.ProjectKey == projectKey {
			count++
			if count > keep {
				continue
			}
		}
		out = append(out, cp)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (s *KnowledgeStore) LatestCheckpoint(project string) (*ContextCheckpoint, error) {
	if strings.TrimSpace(project) == "" {
		return nil, nil
	}
	identity, err := IdentifyProject(project)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	for i := len(st.Checkpoints) - 1; i >= 0; i-- {
		if st.Checkpoints[i].ProjectKey == identity.Key {
			cp := st.Checkpoints[i]
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *KnowledgeStore) SaveRoutine(project string, scope MemoryScope, name, description string, triggers, steps []string, code, language string, tags []string) (Routine, error) {
	if !validMemoryScope(scope) {
		return Routine{}, errors.New("routine scope must be project or global")
	}
	name = cleanText(name, 160)
	description = cleanText(description, 4000)
	code = cleanText(code, 64000)
	language = cleanText(language, 80)
	triggers = cleanList(triggers, 32, 300)
	steps = cleanList(steps, 64, 1200)
	tags = cleanList(tags, 32, 64)
	if name == "" || description == "" {
		return Routine{}, errors.New("routine name and description are required")
	}
	secretText := strings.Join(append(append(append([]string{name, description, code}, triggers...), steps...), tags...), "\n")
	if LooksSecret(secretText) {
		return Routine{}, errors.New("routine appears to contain secret material")
	}
	var identity ProjectIdentity
	var err error
	if scope == MemoryProject {
		identity, err = IdentifyProject(project)
		if err != nil {
			return Routine{}, err
		}
	}
	now := time.Now().UTC()
	routine := Routine{
		ID:          newID("routine"),
		Scope:       scope,
		ProjectKey:  identity.Key,
		ProjectName: identity.Name,
		Name:        name,
		Description: description,
		Triggers:    triggers,
		Steps:       steps,
		Code:        code,
		Language:    language,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadLocked()
	if err != nil {
		return Routine{}, err
	}
	for i := range st.Routines {
		old := st.Routines[i]
		if old.Scope == scope && old.ProjectKey == identity.Key && strings.EqualFold(old.Name, name) {
			routine.ID = old.ID
			routine.CreatedAt = old.CreatedAt
			st.Routines[i] = routine
			if err := s.saveLocked(st); err != nil {
				return Routine{}, err
			}
			return routine, nil
		}
	}
	st.Routines = append(st.Routines, routine)
	if len(st.Routines) > 2000 {
		sort.Slice(st.Routines, func(i, j int) bool { return st.Routines[i].UpdatedAt.After(st.Routines[j].UpdatedAt) })
		st.Routines = st.Routines[:2000]
	}
	if err := s.saveLocked(st); err != nil {
		return Routine{}, err
	}
	return routine, nil
}

func (s *KnowledgeStore) FindRoutines(project, query string, limit int) ([]Routine, error) {
	if limit <= 0 || limit > 30 {
		limit = 8
	}
	var identity ProjectIdentity
	var err error
	if strings.TrimSpace(project) != "" {
		identity, err = IdentifyProject(project)
		if err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return rankRoutines(st.Routines, identity.Key, query, limit), nil
}

func rankRoutines(items []Routine, projectKey, query string, limit int) []Routine {
	tokens := queryTokens(query)
	type scored struct {
		item  Routine
		score int
	}
	var ranked []scored
	for _, item := range items {
		if item.Scope == MemoryProject && (projectKey == "" || item.ProjectKey != projectKey) {
			continue
		}
		score := 0
		if item.Scope == MemoryProject {
			score += 20
		} else {
			score += 12
		}
		if len(tokens) == 0 {
			score++
		} else {
			name := strings.ToLower(item.Name)
			desc := strings.ToLower(item.Description)
			triggerText := strings.ToLower(strings.Join(item.Triggers, " "))
			tags := strings.ToLower(strings.Join(item.Tags, " "))
			steps := strings.ToLower(strings.Join(item.Steps, " "))
			matches := 0
			for _, token := range tokens {
				if strings.Contains(name, token) {
					score += 12
					matches++
				}
				if strings.Contains(triggerText, token) || strings.Contains(tags, token) {
					score += 8
					matches++
				}
				if strings.Contains(desc, token) {
					score += 5
					matches++
				}
				if strings.Contains(steps, token) {
					score += 2
					matches++
				}
			}
			if matches == 0 {
				continue
			}
		}
		ranked = append(ranked, scored{item: item, score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].item.UpdatedAt.After(ranked[j].item.UpdatedAt)
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]Routine, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].item
	}
	return out
}

func (s *KnowledgeStore) BuildContextPack(project, query string, maxItems, maxChars int) (ContextPack, error) {
	if maxItems <= 0 || maxItems > 30 {
		maxItems = 10
	}
	if maxChars <= 0 || maxChars > 48000 {
		maxChars = 16000
	}
	identity, err := IdentifyProject(project)
	if err != nil {
		return ContextPack{}, err
	}
	s.mu.Lock()
	st, err := s.loadLocked()
	s.mu.Unlock()
	if err != nil {
		return ContextPack{}, err
	}
	memories := rankMemories(st.Memories, identity.Key, query, maxItems)
	routines := rankRoutines(st.Routines, identity.Key, query, maxItems/2+1)
	var checkpoint *ContextCheckpoint
	for i := len(st.Checkpoints) - 1; i >= 0; i-- {
		if st.Checkpoints[i].ProjectKey == identity.Key {
			cp := st.Checkpoints[i]
			checkpoint = &cp
			break
		}
	}
	pack := ContextPack{
		Version:     1,
		Project:     identity,
		Query:       cleanText(query, 2000),
		Checkpoint:  checkpoint,
		Memories:    memories,
		Routines:    routines,
		GeneratedAt: time.Now().UTC(),
	}
	pack.ContextText = renderContextPack(pack, maxChars)
	pack.EstimatedChars = len(pack.ContextText)
	return pack, nil
}

func renderContextPack(pack ContextPack, maxChars int) string {
	var b strings.Builder
	appendWithin := func(text string) bool {
		if text == "" {
			return true
		}
		if b.Len()+len(text) > maxChars {
			return false
		}
		b.WriteString(text)
		return true
	}
	appendWithin("WORKBENCH COMPACT CONTEXT\n")
	appendWithin("Project: " + pack.Project.Name + "\n")
	if pack.Checkpoint != nil {
		appendWithin("\nCURRENT CHECKPOINT\n" + pack.Checkpoint.Summary + "\n")
		appendBulletBlock := func(label string, values []string) {
			if len(values) == 0 {
				return
			}
			appendWithin(label + ":\n")
			for _, value := range values {
				if !appendWithin("- " + value + "\n") {
					return
				}
			}
		}
		appendBulletBlock("Decisions", pack.Checkpoint.Decisions)
		appendBulletBlock("Open loops", pack.Checkpoint.OpenLoops)
		appendBulletBlock("Next actions", pack.Checkpoint.NextActions)
	}
	if len(pack.Memories) > 0 {
		appendWithin("\nRELEVANT DURABLE MEMORY\n")
		for _, item := range pack.Memories {
			line := fmt.Sprintf("[%s/%s] %s — %s\n", item.Scope, item.Kind, item.Title, item.Summary)
			if !appendWithin(line) {
				break
			}
			if item.Content != "" {
				content := item.Content
				if len(content) > 1600 {
					content = cleanText(content, 1600) + "…"
				}
				if !appendWithin(content + "\n") {
					break
				}
			}
		}
	}
	if len(pack.Routines) > 0 {
		appendWithin("\nREUSABLE ROUTINES\n")
		for _, routine := range pack.Routines {
			if !appendWithin(fmt.Sprintf("[%s] %s — %s\n", routine.Scope, routine.Name, routine.Description)) {
				break
			}
			for _, step := range routine.Steps {
				if !appendWithin("- " + step + "\n") {
					break
				}
			}
			if routine.Code != "" {
				code := routine.Code
				if len(code) > 4000 {
					code = cleanText(code, 4000) + "…"
				}
				lang := routine.Language
				if lang == "" {
					lang = "text"
				}
				if !appendWithin("```" + lang + "\n" + code + "\n```\n") {
					break
				}
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func (s *KnowledgeStore) RecordTaskOutcome(task Task) error {
	if task.Status != TaskCompleted || strings.TrimSpace(task.ProjectPath) == "" {
		return nil
	}
	summary := cleanText(task.Output, 1400)
	if summary == "" {
		summary = "Task completed successfully."
	}
	if LooksSecret(summary) {
		return nil
	}
	_, err := s.remember(task.ProjectPath, MemoryProject, MemoryOutcome, task.Title, summary, "", []string{"task-outcome"}, "automatic task outcome", task.ID)
	return err
}

func memoryFingerprint(scope MemoryScope, projectKey string, kind MemoryKind, title, summary, content string, tags []string) string {
	parts := []string{string(scope), strings.ToLower(projectKey), string(kind), strings.ToLower(strings.TrimSpace(title)), strings.ToLower(strings.TrimSpace(summary)), strings.TrimSpace(content), strings.Join(tags, "\x1f")}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1e")))
	return hex.EncodeToString(sum[:])
}

func queryTokens(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, token := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len(token) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func cleanText(text string, maxRunes int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
	if maxRunes <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return strings.TrimSpace(string(runes))
}

func cleanList(values []string, maxItems, maxRunes int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = cleanText(value, maxRunes)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
		if len(out) >= maxItems {
			break
		}
	}
	return out
}
