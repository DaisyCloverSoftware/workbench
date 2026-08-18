package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	chatActivityVersion = 1
	chatActivityLimit   = 256
)

type ChatActivityEvent struct {
	ID         string    `json:"id"`
	ProjectRef string    `json:"project_ref"`
	Action     string    `json:"action"`
	State      string    `json:"state"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type chatActivityState struct {
	Version int                 `json:"version"`
	Events  []ChatActivityEvent `json:"events"`
}

var chatActivityMu sync.Mutex

func chatActivityPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "chat-activity.json"), nil
}

func RecordChatActivity(id, projectRef, action, state string) error {
	id = strings.TrimSpace(id)
	projectRef = strings.TrimSpace(projectRef)
	action = strings.ToLower(strings.TrimSpace(action))
	state = strings.ToLower(strings.TrimSpace(state))
	if id == "" || projectRef == "" || action == "" {
		return errors.New("chat activity id, project and action are required")
	}
	if !IsRunnerProjectReference(projectRef) {
		return errors.New("chat activity project must be an opaque runner reference")
	}
	switch state {
	case "running", "completed", "failed", "needs_attention", "waiting":
	default:
		return errors.New("unsupported chat activity state")
	}

	chatActivityMu.Lock()
	defer chatActivityMu.Unlock()

	st, err := loadChatActivityState()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	updated := false
	for i := range st.Events {
		if st.Events[i].ID == id {
			st.Events[i].ProjectRef = projectRef
			st.Events[i].Action = action
			st.Events[i].State = state
			st.Events[i].UpdatedAt = now
			updated = true
			break
		}
	}
	if !updated {
		st.Events = append(st.Events, ChatActivityEvent{ID: id, ProjectRef: projectRef, Action: action, State: state, UpdatedAt: now})
	}
	pruneChatActivity(&st, now)
	return saveChatActivityState(st)
}

func RecentChatActivity(limit int, since time.Time) ([]ChatActivityEvent, error) {
	chatActivityMu.Lock()
	defer chatActivityMu.Unlock()
	st, err := loadChatActivityState()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > chatActivityLimit {
		limit = 64
	}
	items := make([]ChatActivityEvent, 0, len(st.Events))
	for _, event := range st.Events {
		if !since.IsZero() && event.UpdatedAt.Before(since) {
			continue
		}
		items = append(items, event)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func loadChatActivityState() (chatActivityState, error) {
	path, err := chatActivityPath()
	if err != nil {
		return chatActivityState{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return chatActivityState{Version: chatActivityVersion}, nil
	}
	if err != nil {
		return chatActivityState{}, err
	}
	var st chatActivityState
	if err := json.Unmarshal(b, &st); err != nil {
		return chatActivityState{}, err
	}
	if st.Version == 0 {
		st.Version = chatActivityVersion
	}
	return st, nil
}

func saveChatActivityState(st chatActivityState) error {
	path, err := chatActivityPath()
	if err != nil {
		return err
	}
	st.Version = chatActivityVersion
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

func pruneChatActivity(st *chatActivityState, now time.Time) {
	cutoff := now.Add(-7 * 24 * time.Hour)
	kept := st.Events[:0]
	for _, event := range st.Events {
		if event.UpdatedAt.Before(cutoff) {
			continue
		}
		kept = append(kept, event)
	}
	st.Events = kept
	sort.SliceStable(st.Events, func(i, j int) bool { return st.Events[i].UpdatedAt.After(st.Events[j].UpdatedAt) })
	if len(st.Events) > chatActivityLimit {
		st.Events = st.Events[:chatActivityLimit]
	}
}
