package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RelayRecord links a transport-level relay message to the durable Workbench task
// that was created from it. The relay process writes this file; the MCP server
// only reads it, so personal ChatGPT plans can use a read-only custom MCP to
// inspect status while GitHub provides the supported write transport.
type RelayRecord struct {
	RelayID         string    `json:"relay_id"`
	Source          string    `json:"source"`
	SourcePath      string    `json:"source_path"`
	WorkbenchTaskID string    `json:"workbench_task_id,omitempty"`
	Project         string    `json:"project,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Error           string    `json:"error,omitempty"`
}

type relayState struct {
	Version int                    `json:"version"`
	Records map[string]RelayRecord `json:"records"`
}

func RelayStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// os.UserConfigDir is ~/.config on Linux. Relay state is operational rather
	// than secret, but keep the file private because task titles may be sensitive.
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
		return relayState{Version: 1, Records: map[string]RelayRecord{}}, nil
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
	if st.Version == 0 {
		st.Version = 1
	}
	return st, nil
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
	if rec.CreatedAt.IsZero() {
		if old, ok := st.Records[rec.RelayID]; ok && !old.CreatedAt.IsZero() {
			rec.CreatedAt = old.CreatedAt
		} else {
			rec.CreatedAt = now
		}
	}
	rec.UpdatedAt = now
	st.Records[rec.RelayID] = rec
	path, err := RelayStatePath()
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

func LoadRelayRecord(id string) (RelayRecord, bool, error) {
	st, err := loadRelayState()
	if err != nil {
		return RelayRecord{}, false, err
	}
	rec, ok := st.Records[strings.TrimSpace(id)]
	return rec, ok, nil
}
