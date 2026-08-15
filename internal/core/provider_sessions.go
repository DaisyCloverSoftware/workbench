package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	providerSessionStoreVersion = 1
	maxProviderSessionStoreBytes = 1 << 20
	maxProviderSessionIDBytes     = 512
)

var (
	providerSessionMu           sync.RWMutex
	providerSessionPathOverride string
)

type ProviderSession struct {
	TaskID     string    `json:"task_id"`
	ProviderID string    `json:"provider_id"`
	SessionID  string    `json:"session_id"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type providerSessionState struct {
	Version  int               `json:"version"`
	Sessions []ProviderSession `json:"sessions"`
}

func ProviderSessionStatePath() (string, error) {
	if override := strings.TrimSpace(providerSessionPathOverride); override != "" {
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
	return filepath.Join(dir, "provider-sessions.json"), nil
}

func ProviderSessionFor(taskID, providerID string) (ProviderSession, bool, error) {
	taskID = strings.TrimSpace(taskID)
	providerID = strings.TrimSpace(providerID)
	if taskID == "" || providerID == "" {
		return ProviderSession{}, false, nil
	}
	providerSessionMu.RLock()
	defer providerSessionMu.RUnlock()
	st, err := loadProviderSessionStateUnlocked()
	if err != nil {
		return ProviderSession{}, false, err
	}
	for _, session := range st.Sessions {
		if session.TaskID == taskID && session.ProviderID == providerID {
			return session, true, nil
		}
	}
	return ProviderSession{}, false, nil
}

func SaveProviderSession(taskID, providerID, sessionID string) (ProviderSession, error) {
	taskID = strings.TrimSpace(taskID)
	providerID = strings.TrimSpace(providerID)
	sessionID = strings.TrimSpace(sessionID)
	if taskID == "" || providerID == "" {
		return ProviderSession{}, errors.New("provider session requires task and provider identity")
	}
	if err := validateProviderSessionID(sessionID); err != nil {
		return ProviderSession{}, err
	}
	release, err := lockProviderSessionWrite()
	if err != nil {
		return ProviderSession{}, err
	}
	defer release()
	st, err := loadProviderSessionStateUnlocked()
	if err != nil {
		return ProviderSession{}, err
	}
	saved := ProviderSession{TaskID: taskID, ProviderID: providerID, SessionID: sessionID, UpdatedAt: time.Now().UTC()}
	updated := false
	for i := range st.Sessions {
		if st.Sessions[i].TaskID == taskID && st.Sessions[i].ProviderID == providerID {
			st.Sessions[i] = saved
			updated = true
			break
		}
	}
	if !updated {
		st.Sessions = append(st.Sessions, saved)
	}
	if err := saveProviderSessionStateUnlocked(st); err != nil {
		return ProviderSession{}, err
	}
	return saved, nil
}

func DeleteProviderSession(taskID, providerID string) error {
	taskID = strings.TrimSpace(taskID)
	providerID = strings.TrimSpace(providerID)
	if taskID == "" || providerID == "" {
		return nil
	}
	release, err := lockProviderSessionWrite()
	if err != nil {
		return err
	}
	defer release()
	st, err := loadProviderSessionStateUnlocked()
	if err != nil {
		return err
	}
	out := st.Sessions[:0]
	for _, session := range st.Sessions {
		if session.TaskID == taskID && session.ProviderID == providerID {
			continue
		}
		out = append(out, session)
	}
	st.Sessions = out
	return saveProviderSessionStateUnlocked(st)
}

func DeleteTaskProviderSessions(taskID string) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil
	}
	release, err := lockProviderSessionWrite()
	if err != nil {
		return err
	}
	defer release()
	st, err := loadProviderSessionStateUnlocked()
	if err != nil {
		return err
	}
	out := st.Sessions[:0]
	for _, session := range st.Sessions {
		if session.TaskID == taskID {
			continue
		}
		out = append(out, session)
	}
	st.Sessions = out
	return saveProviderSessionStateUnlocked(st)
}

func validateProviderSessionID(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("provider session id is empty")
	}
	if len(sessionID) > maxProviderSessionIDBytes {
		return errors.New("provider session id is unexpectedly long")
	}
	for _, r := range sessionID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return errors.New("provider session id contains unsupported characters")
	}
	return nil
}

func loadProviderSessionStateUnlocked() (providerSessionState, error) {
	path, err := ProviderSessionStatePath()
	if err != nil {
		return providerSessionState{}, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return providerSessionState{Version: providerSessionStoreVersion}, nil
	}
	if err != nil {
		return providerSessionState{}, err
	}
	if !info.Mode().IsRegular() {
		return providerSessionState{}, errors.New("provider session store is not a regular file")
	}
	if info.Size() > maxProviderSessionStoreBytes {
		return providerSessionState{}, errors.New("provider session store is unexpectedly large")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return providerSessionState{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	var st providerSessionState
	if err := dec.Decode(&st); err != nil {
		return providerSessionState{}, err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return providerSessionState{}, errors.New("provider session store contains more than one JSON value")
		}
		return providerSessionState{}, err
	}
	if st.Version != providerSessionStoreVersion {
		return providerSessionState{}, fmt.Errorf("unsupported provider session store version %d", st.Version)
	}
	seen := make(map[string]bool, len(st.Sessions))
	for i := range st.Sessions {
		session := &st.Sessions[i]
		session.TaskID = strings.TrimSpace(session.TaskID)
		session.ProviderID = strings.TrimSpace(session.ProviderID)
		session.SessionID = strings.TrimSpace(session.SessionID)
		if session.TaskID == "" || session.ProviderID == "" {
			return providerSessionState{}, errors.New("provider session store contains an entry without task/provider identity")
		}
		if err := validateProviderSessionID(session.SessionID); err != nil {
			return providerSessionState{}, err
		}
		key := session.TaskID + "\x00" + session.ProviderID
		if seen[key] {
			return providerSessionState{}, errors.New("provider session store contains duplicate task/provider entries")
		}
		seen[key] = true
	}
	return st, nil
}

func saveProviderSessionStateUnlocked(st providerSessionState) error {
	path, err := ProviderSessionStatePath()
	if err != nil {
		return err
	}
	st.Version = providerSessionStoreVersion
	body, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if len(body) > maxProviderSessionStoreBytes {
		return errors.New("provider session store exceeds its local size limit")
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".provider-sessions-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(body); err != nil {
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

func lockProviderSessionWrite() (func(), error) {
	providerSessionMu.Lock()
	path, err := ProviderSessionStatePath()
	if err != nil {
		providerSessionMu.Unlock()
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
				providerSessionMu.Unlock()
			}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			providerSessionMu.Unlock()
			return nil, openErr
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 2*time.Minute {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			providerSessionMu.Unlock()
			return nil, errors.New("timed out waiting for Workbench provider-session lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
