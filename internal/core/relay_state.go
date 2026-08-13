package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RelayRecord links a transport-level relay task message to the durable
// Workbench task that was created from it.
type RelayRecord struct {
	RelayID          string    `json:"relay_id"`
	Source           string    `json:"source"`
	SourcePath       string    `json:"source_path"`
	WorkbenchTaskID  string    `json:"workbench_task_id,omitempty"`
	Project          string    `json:"project,omitempty"`
	LastAnswerDigest string    `json:"last_answer_digest,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Error            string    `json:"error,omitempty"`
}

// RelayControlRecord makes private relay control envelopes idempotent. Response
// contains only the already-filtered model-facing MCP result that can be written
// back to a verified private relay transport.
type RelayControlRecord struct {
	RelayID    string          `json:"relay_id"`
	SourcePath string          `json:"source_path"`
	Digest     string          `json:"digest"`
	Action     string          `json:"action"`
	Response   json.RawMessage `json:"response"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type relayState struct {
	Version  int                           `json:"version"`
	Records  map[string]RelayRecord        `json:"records"`
	Controls map[string]RelayControlRecord `json:"controls,omitempty"`
}

func RelayStatePath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("WORKBENCH_RELAY_STATE_PATH")); override != "" {
		path, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return "", err
		}
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "github-relay.json"), nil
}

func loadRelayState() (relayState, error) {
	path, err := RelayStatePath()
	if err != nil {
		return relayState{}, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return relayState{Version: 2, Records: map[string]RelayRecord{}, Controls: map[string]RelayControlRecord{}}, nil
	}
	if err != nil {
		return relayState{}, err
	}
	var st relayState
	if err := json.Unmarshal(b, &st); err != nil {
		return relayState{}, err
	}
	if st.Records == nil {
		st.Records = map[string]RelayRecord{}
	}
	if st.Controls == nil {
		st.Controls = map[string]RelayControlRecord{}
	}
	if st.Version < 2 {
		st.Version = 2
	}
	return st, nil
}

func saveRelayState(st relayState) error {
	path, err := RelayStatePath()
	if err != nil {
		return err
	}
	st.Version = 2
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

func SaveRelayRecord(rec RelayRecord) error {
	rec.RelayID = strings.TrimSpace(rec.RelayID)
	if rec.RelayID == "" {
		return errors.New("relay id is empty")
	}
	st, err := loadRelayState()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if old, ok := st.Records[rec.RelayID]; ok {
		if rec.CreatedAt.IsZero() && !old.CreatedAt.IsZero() {
			rec.CreatedAt = old.CreatedAt
		}
		if rec.LastAnswerDigest == "" {
			rec.LastAnswerDigest = old.LastAnswerDigest
		}
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	st.Records[rec.RelayID] = rec
	return saveRelayState(st)
}

func LoadRelayRecord(id string) (RelayRecord, bool, error) {
	st, err := loadRelayState()
	if err != nil {
		return RelayRecord{}, false, err
	}
	rec, ok := st.Records[strings.TrimSpace(id)]
	return rec, ok, nil
}

func ListRelayRecords() ([]RelayRecord, error) {
	st, err := loadRelayState()
	if err != nil {
		return nil, err
	}
	out := make([]RelayRecord, 0, len(st.Records))
	for _, rec := range st.Records {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelayID < out[j].RelayID })
	return out, nil
}

func SaveRelayControlRecord(rec RelayControlRecord) error {
	rec.RelayID = strings.TrimSpace(rec.RelayID)
	rec.Digest = strings.TrimSpace(rec.Digest)
	rec.Action = strings.TrimSpace(rec.Action)
	if rec.RelayID == "" || rec.Digest == "" || rec.Action == "" {
		return errors.New("relay control id, digest and action are required")
	}
	st, err := loadRelayState()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if old, ok := st.Controls[rec.RelayID]; ok && rec.CreatedAt.IsZero() {
		rec.CreatedAt = old.CreatedAt
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	st.Controls[rec.RelayID] = rec
	return saveRelayState(st)
}

func LoadRelayControlRecord(id string) (RelayControlRecord, bool, error) {
	st, err := loadRelayState()
	if err != nil {
		return RelayControlRecord{}, false, err
	}
	rec, ok := st.Controls[strings.TrimSpace(id)]
	return rec, ok, nil
}

func ListRelayControlRecords() ([]RelayControlRecord, error) {
	st, err := loadRelayState()
	if err != nil {
		return nil, err
	}
	out := make([]RelayControlRecord, 0, len(st.Controls))
	for _, rec := range st.Controls {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelayID < out[j].RelayID })
	return out, nil
}
