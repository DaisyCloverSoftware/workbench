package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type KnowledgeScope string

type KnowledgeKind string

const (
	ScopeGlobal  KnowledgeScope = "global"
	ScopeProject KnowledgeScope = "project"

	KindFact       KnowledgeKind = "fact"
	KindDecision   KnowledgeKind = "decision"
	KindConstraint KnowledgeKind = "constraint"
	KindPattern    KnowledgeKind = "pattern"
	KindRoutine    KnowledgeKind = "routine"
	KindCode       KnowledgeKind = "code"
)

type KnowledgeItem struct {
	ID          string         `json:"id"`
	Scope       KnowledgeScope `json:"scope"`
	Project     string         `json:"project,omitempty"`
	Kind        KnowledgeKind  `json:"kind"`
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Tags        []string       `json:"tags,omitempty"`
	Source      string         `json:"source,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	UseCount    int            `json:"use_count,omitempty"`
	LastUsedAt  *time.Time     `json:"last_used_at,omitempty"`
	Fingerprint string         `json:"fingerprint,omitempty"`
}

type ContextCapsule struct {
	ID          string    `json:"id"`
	Project     string    `json:"project,omitempty"`
	Objective   string    `json:"objective"`
	State       string    `json:"state"`
	Decisions   []string  `json:"decisions,omitempty"`
	Constraints []string  `json:"constraints,omitempty"`
	References  []string  `json:"references,omitempty"`
	OpenThreads []string  `json:"open_threads,omitempty"`
	NextAction  string    `json:"next_action,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type knowledgeState struct {
	Version  int              `json:"version"`
	Items    []KnowledgeItem  `json:"items"`
	Capsules []ContextCapsule `json:"capsules"`
}

func KnowledgeStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "knowledge.json"), nil
}

func loadKnowledgeState() (knowledgeState, error) {
	path, err := KnowledgeStatePath()
	if err != nil {
		return knowledgeState{}, err
	}
	b, err := os.ReadFile(path)
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

func saveKnowledgeState(st knowledgeState) error {
	path, err := KnowledgeStatePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func SaveKnowledge(item KnowledgeItem) (KnowledgeItem, error) {
	item.Title = strings.TrimSpace(item.Title)
	item.Content = strings.TrimSpace(item.Content)
	item.Project = strings.TrimSpace(item.Project)
	item.Source = strings.TrimSpace(item.Source)
	if item.Title == "" || item.Content == "" {
		return KnowledgeItem{}, errors.New("knowledge title and content are required")
	}
	if LooksSecret(item.Title + "\n" + item.Content) {
		return KnowledgeItem{}, errors.New("knowledge appears to contain secret material; store secrets in the vault")
	}
	if item.Scope != ScopeGlobal && item.Scope != ScopeProject {
		return KnowledgeItem{}, errors.New("knowledge scope must be global or project")
	}
	if item.Scope == ScopeProject && item.Project == "" {
		return KnowledgeItem{}, errors.New("project-scoped knowledge requires a project")
	}
	if item.Kind == "" {
		item.Kind = KindFact
	}
	item.Tags = normalizeTags(item.Tags)
	if item.Fingerprint == "" {
		item.Fingerprint = knowledgeFingerprint(item)
	}

	release, err := lockKnowledgeWrite()
	if err != nil {
		return KnowledgeItem{}, err
	}
	defer release()

	st, err := loadKnowledgeState()
	if err != nil {
		return KnowledgeItem{}, err
	}
	now := time.Now().UTC()
	for i := range st.Items {
		old := &st.Items[i]
		if (item.ID != "" && old.ID == item.ID) || (item.ID == "" && old.Fingerprint == item.Fingerprint) {
			item.ID = old.ID
			item.CreatedAt = old.CreatedAt
			item.UseCount = old.UseCount
			item.LastUsedAt = old.LastUsedAt
			item.UpdatedAt = now
			st.Items[i] = item
			if err := saveKnowledgeState(st); err != nil {
				return KnowledgeItem{}, err
			}
			return item, nil
		}
	}
	if item.ID == "" {
		item.ID = newID("mem")
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	st.Items = append(st.Items, item)
	if err := saveKnowledgeState(st); err != nil {
		return KnowledgeItem{}, err
	}
	return item, nil
}

func SearchKnowledge(project, query string, limit int) ([]KnowledgeItem, error) {
	knowledgeMu.RLock()
	defer knowledgeMu.RUnlock()

	st, err := loadKnowledgeState()
	if err != nil {
		return nil, err
	}
	project = strings.TrimSpace(project)
	terms := searchTerms(query)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	type scored struct {
		item  KnowledgeItem
		score int
	}
	var found []scored
	for _, item := range st.Items {
		if item.Scope == ScopeProject && item.Project != project {
			continue
		}
		hay := strings.ToLower(item.Title + "\n" + item.Content + "\n" + strings.Join(item.Tags, " "))
		score := 0
		for _, term := range terms {
			if strings.Contains(hay, term) {
				score += 3
			}
			if strings.Contains(strings.ToLower(item.Title), term) {
				score += 3
			}
		}
		if len(terms) == 0 {
			score = 1
		}
		if item.Project == project && project != "" {
			score++
		}
		if score > 0 {
			found = append(found, scored{item: item, score: score})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].item.UpdatedAt.After(found[j].item.UpdatedAt)
	})
	capacity := limit
	if len(found) < capacity {
		capacity = len(found)
	}
	out := make([]KnowledgeItem, 0, capacity)
	for i := 0; i < len(found) && i < limit; i++ {
		out = append(out, found[i].item)
	}
	return out, nil
}

func MarkKnowledgeUsed(id string) error {
	release, err := lockKnowledgeWrite()
	if err != nil {
		return err
	}
	defer release()

	st, err := loadKnowledgeState()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range st.Items {
		if st.Items[i].ID == strings.TrimSpace(id) {
			st.Items[i].UseCount++
			st.Items[i].LastUsedAt = &now
			st.Items[i].UpdatedAt = now
			return saveKnowledgeState(st)
		}
	}
	return errors.New("knowledge item not found")
}

func SaveContextCapsule(c ContextCapsule) (ContextCapsule, error) {
	c.Objective = strings.TrimSpace(c.Objective)
	c.State = strings.TrimSpace(c.State)
	c.Project = strings.TrimSpace(c.Project)
	if c.Objective == "" || c.State == "" {
		return ContextCapsule{}, errors.New("context capsule objective and state are required")
	}
	combined := c.Objective + "\n" + c.State + "\n" + strings.Join(c.Decisions, "\n") + "\n" + strings.Join(c.Constraints, "\n") + "\n" + c.NextAction
	if LooksSecret(combined) {
		return ContextCapsule{}, errors.New("context capsule appears to contain secret material")
	}

	release, err := lockKnowledgeWrite()
	if err != nil {
		return ContextCapsule{}, err
	}
	defer release()

	st, err := loadKnowledgeState()
	if err != nil {
		return ContextCapsule{}, err
	}
	now := time.Now().UTC()
	if c.ID == "" {
		c.ID = newID("ctx")
		c.CreatedAt = now
	} else {
		for _, old := range st.Capsules {
			if old.ID == c.ID && !old.CreatedAt.IsZero() {
				c.CreatedAt = old.CreatedAt
				break
			}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = now
		}
	}
	c.UpdatedAt = now
	replaced := false
	for i := range st.Capsules {
		if st.Capsules[i].ID == c.ID {
			st.Capsules[i] = c
			replaced = true
			break
		}
	}
	if !replaced {
		st.Capsules = append(st.Capsules, c)
	}
	if err := saveKnowledgeState(st); err != nil {
		return ContextCapsule{}, err
	}
	return c, nil
}

func LatestContextCapsule(project string) (ContextCapsule, bool, error) {
	knowledgeMu.RLock()
	defer knowledgeMu.RUnlock()

	st, err := loadKnowledgeState()
	if err != nil {
		return ContextCapsule{}, false, err
	}
	project = strings.TrimSpace(project)
	var best ContextCapsule
	for _, c := range st.Capsules {
		if project != "" && c.Project != project {
			continue
		}
		if best.ID == "" || c.UpdatedAt.After(best.UpdatedAt) {
			best = c
		}
	}
	return best, best.ID != "", nil
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func searchTerms(q string) []string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(q)))
	seen := map[string]bool{}
	out := parts[:0]
	for _, p := range parts {
		if len(p) < 2 || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func knowledgeFingerprint(item KnowledgeItem) string {
	base := strings.ToLower(string(item.Scope) + "\n" + item.Project + "\n" + string(item.Kind) + "\n" + item.Title + "\n" + item.Content)
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}
