package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	assetTagPrefix     = "workbench-asset:"
	assetTagActive     = assetTagPrefix + "active"
	assetTagSuperseded = assetTagPrefix + "superseded"
	assetTagVerified   = assetTagPrefix + "verified"
	assetVersionPrefix = assetTagPrefix + "v"
)

type ReusableAssetMeta struct {
	KnowledgeID  string    `json:"knowledge_id"`
	Lineage      string    `json:"lineage"`
	Version      int       `json:"version"`
	Status       string    `json:"status"`
	Supersedes   string    `json:"supersedes,omitempty"`
	Verification string    `json:"verification,omitempty"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type reusableAssetState struct {
	Version int                 `json:"version"`
	Items   []ReusableAssetMeta `json:"items"`
}

var reusableAssetMu sync.Mutex

func ReusableAssetStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "reusable-assets.json"), nil
}

func loadReusableAssetState() (reusableAssetState, error) {
	path, err := ReusableAssetStatePath()
	if err != nil {
		return reusableAssetState{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return reusableAssetState{Version: 1}, nil
	}
	if err != nil {
		return reusableAssetState{}, err
	}
	var st reusableAssetState
	if err := json.Unmarshal(b, &st); err != nil {
		return reusableAssetState{}, err
	}
	if st.Version == 0 {
		st.Version = 1
	}
	return st, nil
}

func saveReusableAssetState(st reusableAssetState) error {
	path, err := ReusableAssetStatePath()
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

func SaveReusableAsset(item KnowledgeItem, verification string) (KnowledgeItem, ReusableAssetMeta, error) {
	if item.Kind != KindRoutine && item.Kind != KindCode {
		return KnowledgeItem{}, ReusableAssetMeta{}, errors.New("reusable assets must be routine or code knowledge")
	}
	verification = strings.TrimSpace(verification)
	if LooksSecret(verification) {
		return KnowledgeItem{}, ReusableAssetMeta{}, errors.New("asset verification appears to contain secret material")
	}

	reusableAssetMu.Lock()
	defer reusableAssetMu.Unlock()

	st, err := loadReusableAssetState()
	if err != nil {
		return KnowledgeItem{}, ReusableAssetMeta{}, err
	}
	candidates, err := SearchKnowledge(item.Project, item.Title, 100)
	if err != nil {
		return KnowledgeItem{}, ReusableAssetMeta{}, err
	}
	lineage := reusableAssetLineage(item)
	fingerprint := knowledgeFingerprint(item)

	var exact *KnowledgeItem
	var latest *KnowledgeItem
	latestVersion := 0
	for i := range candidates {
		candidate := candidates[i]
		if !sameReusableAsset(candidate, item) {
			continue
		}
		version := ReusableAssetVersion(candidate)
		if candidate.Fingerprint == fingerprint {
			copy := candidate
			exact = &copy
		}
		if latest == nil || version > latestVersion {
			copy := candidate
			latest = &copy
			latestVersion = version
		}
	}

	now := time.Now().UTC()
	if exact != nil {
		meta := reusableAssetMetaForState(st, exact.ID)
		if meta.KnowledgeID == "" {
			meta = ReusableAssetMeta{KnowledgeID: exact.ID, Lineage: lineage, Version: ReusableAssetVersion(*exact), Status: "active", CreatedAt: exact.CreatedAt}
		}
		if verification != "" {
			meta.Verification = verification
			meta.VerifiedAt = &now
		}
		meta.UpdatedAt = now
		*exact = applyReusableAssetTags(*exact, meta)
		saved, err := SaveKnowledge(*exact)
		if err != nil {
			return KnowledgeItem{}, ReusableAssetMeta{}, err
		}
		upsertReusableAssetMeta(&st, meta)
		if err := saveReusableAssetState(st); err != nil {
			return KnowledgeItem{}, ReusableAssetMeta{}, err
		}
		return saved, meta, nil
	}

	meta := ReusableAssetMeta{Lineage: lineage, Version: latestVersion + 1, Status: "active", CreatedAt: now, UpdatedAt: now}
	if meta.Version <= 0 {
		meta.Version = 1
	}
	if verification != "" {
		meta.Verification = verification
		meta.VerifiedAt = &now
	}
	if latest != nil {
		meta.Supersedes = latest.ID
		oldMeta := reusableAssetMetaForState(st, latest.ID)
		if oldMeta.KnowledgeID == "" {
			oldMeta = ReusableAssetMeta{KnowledgeID: latest.ID, Lineage: lineage, Version: ReusableAssetVersion(*latest), CreatedAt: latest.CreatedAt}
		}
		oldMeta.Status = "superseded"
		oldMeta.UpdatedAt = now
		*latest = applyReusableAssetTags(*latest, oldMeta)
		if _, err := SaveKnowledge(*latest); err != nil {
			return KnowledgeItem{}, ReusableAssetMeta{}, err
		}
		upsertReusableAssetMeta(&st, oldMeta)
	}

	item = applyReusableAssetTags(item, meta)
	saved, err := SaveKnowledge(item)
	if err != nil {
		return KnowledgeItem{}, ReusableAssetMeta{}, err
	}
	meta.KnowledgeID = saved.ID
	item = applyReusableAssetTags(saved, meta)
	if _, err := SaveKnowledge(item); err != nil {
		return KnowledgeItem{}, ReusableAssetMeta{}, err
	}
	upsertReusableAssetMeta(&st, meta)
	if err := saveReusableAssetState(st); err != nil {
		return KnowledgeItem{}, ReusableAssetMeta{}, err
	}
	return item, meta, nil
}

// FilterActiveReusableKnowledge removes superseded routine/code revisions while
// preserving the caller's ordering. SearchKnowledge already owns relevance
// ranking; a filter must not silently reshuffle lower-relevance memories ahead
// of task-specific ones.
func FilterActiveReusableKnowledge(items []KnowledgeItem) []KnowledgeItem {
	out := make([]KnowledgeItem, 0, len(items))
	for _, item := range items {
		if (item.Kind == KindRoutine || item.Kind == KindCode) && hasTag(item.Tags, assetTagSuperseded) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func ReusableAssetVersion(item KnowledgeItem) int {
	for _, tag := range item.Tags {
		if strings.HasPrefix(tag, assetVersionPrefix) {
			if n, err := strconv.Atoi(strings.TrimPrefix(tag, assetVersionPrefix)); err == nil && n > 0 {
				return n
			}
	}
	if item.Kind == KindRoutine || item.Kind == KindCode {
		return 1
	}
	return 0
}

func ReusableAssetVerified(item KnowledgeItem) bool {
	return hasTag(item.Tags, assetTagVerified)
}

func ReusableAssetMetadata(id string) (ReusableAssetMeta, bool, error) {
	reusableAssetMu.Lock()
	defer reusableAssetMu.Unlock()
	st, err := loadReusableAssetState()
	if err != nil {
		return ReusableAssetMeta{}, false, err
	}
	meta := reusableAssetMetaForState(st, strings.TrimSpace(id))
	return meta, meta.KnowledgeID != "", nil
}

func reusableAssetLineage(item KnowledgeItem) string {
	return strings.ToLower(string(item.Scope) + "\n" + strings.TrimSpace(item.Project) + "\n" + string(item.Kind) + "\n" + strings.TrimSpace(item.Title))
}

func sameReusableAsset(a, b KnowledgeItem) bool {
	return a.Scope == b.Scope && strings.TrimSpace(a.Project) == strings.TrimSpace(b.Project) && a.Kind == b.Kind && strings.EqualFold(strings.TrimSpace(a.Title), strings.TrimSpace(b.Title))
}

func applyReusableAssetTags(item KnowledgeItem, meta ReusableAssetMeta) KnowledgeItem {
	tags := make([]string, 0, len(item.Tags)+3)
	for _, tag := range item.Tags {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(tag)), assetTagPrefix) {
			tags = append(tags, tag)
		}
	}
	if meta.Status == "superseded" {
		tags = append(tags, assetTagSuperseded)
	} else {
		tags = append(tags, assetTagActive)
	}
	tags = append(tags, assetVersionPrefix+strconv.Itoa(max(1, meta.Version)))
	if meta.VerifiedAt != nil {
		tags = append(tags, assetTagVerified)
	}
	item.Tags = normalizeTags(tags)
	return item
}

func reusableAssetMetaForState(st reusableAssetState, id string) ReusableAssetMeta {
	for _, meta := range st.Items {
		if meta.KnowledgeID == id {
			return meta
		}
	}
	return ReusableAssetMeta{}
}

func upsertReusableAssetMeta(st *reusableAssetState, meta ReusableAssetMeta) {
	for i := range st.Items {
		if st.Items[i].KnowledgeID == meta.KnowledgeID {
			st.Items[i] = meta
			return
		}
	}
	st.Items = append(st.Items, meta)
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), want) {
			return true
		}
	}
	return false
}
