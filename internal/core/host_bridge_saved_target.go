package core

import (
	"errors"
	"strings"
	"sync"
)

// SavedHostBridgeTarget separates the bridge's connection authority from
// preferences tentatively changed in memory before a save reaches disk.
// Readers remain available while persistence runs. It grants no tool authority;
// the Windows cycle still validates the target and every typed host job.
type SavedHostBridgeTarget struct {
	saveMu sync.Mutex
	mu     sync.RWMutex
	host   string
}

func NewSavedHostBridgeTarget(initial string) *SavedHostBridgeTarget {
	return &SavedHostBridgeTarget{host: strings.TrimSpace(initial)}
}

func (t *SavedHostBridgeTarget) Current() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.host
}

// Save publishes host only after persist succeeds. Serialize concurrent saves
// without blocking Current, and never cancel or redirect an in-flight cycle.
func (t *SavedHostBridgeTarget) Save(host string, persist func() error) error {
	if persist == nil {
		return errors.New("host bridge target requires a persistence callback")
	}
	t.saveMu.Lock()
	defer t.saveMu.Unlock()
	if err := persist(); err != nil {
		return err
	}
	t.mu.Lock()
	t.host = strings.TrimSpace(host)
	t.mu.Unlock()
	return nil
}
