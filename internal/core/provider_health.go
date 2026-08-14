package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ProviderHealth struct {
	ProviderID          string    `json:"provider_id"`
	Reason              string    `json:"reason"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastFailureAt       time.Time `json:"last_failure_at"`
	CooldownUntil       time.Time `json:"cooldown_until"`
}

type providerHealthState struct {
	Version int              `json:"version"`
	Entries []ProviderHealth `json:"entries"`
}

const (
	providerHealthStateVersion = 1
	maxProviderHealthBytes     = 256 << 10
)

var providerHealthMu sync.Mutex

func ProviderHealthStatePath() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "provider-health.json"), nil
}

// ApplyProviderHealth adds safe, categorical cooldown telemetry to provider
// status text. Health persistence deliberately stores no raw worker output,
// credentials, account identifiers or task content.
func ApplyProviderHealth(providers []Provider, now time.Time) []Provider {
	out := append([]Provider(nil), providers...)
	st, err := loadProviderHealthState()
	if err != nil {
		return out
	}
	active := activeProviderHealth(st, now)
	for i := range out {
		record, ok := active[out[i].ID]
		if !ok {
			continue
		}
		remaining := record.CooldownUntil.Sub(now)
		if remaining < time.Second {
			remaining = time.Second
		}
		status := strings.TrimSpace(out[i].Status)
		if status != "" {
			status += " · "
		}
		out[i].Status = status + fmt.Sprintf("cooldown %s · %s", compactCooldown(remaining), record.Reason)
	}
	return out
}

// FilterProviderCooldowns removes providers whose recent provider-level failure
// is still inside its short-lived cooldown. Cache read failures fail open so
// health telemetry can never prevent useful work.
func FilterProviderCooldowns(providers []Provider, now time.Time) ([]Provider, []string) {
	st, err := loadProviderHealthState()
	if err != nil {
		return append([]Provider(nil), providers...), nil
	}
	active := activeProviderHealth(st, now)
	ready := make([]Provider, 0, len(providers))
	var skipped []string
	for _, p := range providers {
		record, cooling := active[p.ID]
		if !cooling {
			ready = append(ready, p)
			continue
		}
		skipped = append(skipped, fmt.Sprintf("%s: skipped until %s (%s)", p.Name, record.CooldownUntil.UTC().Format(time.RFC3339), record.Reason))
	}
	return ready, skipped
}

// RecordProviderRunOutcome updates best-effort local health telemetry. Only
// retryable provider/setup failures create cooldowns; task-specific coding
// failures do not poison a provider globally. A successful run clears the
// provider's prior cooldown immediately.
func RecordProviderRunOutcome(providerID string, res RunResult, runErr error) (ProviderHealth, bool) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return ProviderHealth{}, false
	}
	if runErr == nil {
		_ = ClearProviderHealth(providerID)
		return ProviderHealth{}, false
	}
	if !res.Retryable {
		return ProviderHealth{}, false
	}
	record, err := recordProviderRetryableFailureAt(providerID, res, runErr, time.Now().UTC())
	return record, err == nil
}

func ClearProviderHealth(providerID string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil
	}
	return mutateProviderHealthState(func(st *providerHealthState) {
		out := st.Entries[:0]
		for _, entry := range st.Entries {
			if entry.ProviderID != providerID {
				out = append(out, entry)
			}
		}
		st.Entries = out
	})
}

// ClearAllProviderCooldowns is intended for an explicit operator rescan after
// fixing a login/tool setup issue. Automatic task routing never clears another
// provider's cooldown merely because a scan ran.
func ClearAllProviderCooldowns() error {
	return mutateProviderHealthState(func(st *providerHealthState) {
		st.Entries = nil
	})
}

func recordProviderRetryableFailureAt(providerID string, res RunResult, runErr error, now time.Time) (ProviderHealth, error) {
	reason, base := providerCooldownReason(res, runErr)
	var recorded ProviderHealth
	err := mutateProviderHealthState(func(st *providerHealthState) {
		count := 1
		for _, entry := range st.Entries {
			if entry.ProviderID == providerID && !entry.LastFailureAt.IsZero() && now.Sub(entry.LastFailureAt) <= 30*time.Minute {
				count = entry.ConsecutiveFailures + 1
				break
			}
		}
		duration := backoffCooldown(base, count)
		recorded = ProviderHealth{
			ProviderID:          providerID,
			Reason:              reason,
			ConsecutiveFailures: count,
			LastFailureAt:       now.UTC(),
			CooldownUntil:       now.Add(duration).UTC(),
		}
		out := st.Entries[:0]
		for _, entry := range st.Entries {
			if entry.ProviderID != providerID && (entry.LastFailureAt.IsZero() || now.Sub(entry.LastFailureAt) < 24*time.Hour) {
				out = append(out, entry)
			}
		}
		st.Entries = append(out, recorded)
	})
	return recorded, err
}

func providerCooldownReason(res RunResult, runErr error) (string, time.Duration) {
	low := strings.ToLower(strings.Join([]string{res.WorkerUnavailable, res.Attention, res.Output, errorText(runErr)}, " "))
	switch {
	case res.Authentication || containsAny(low, "unauthorized", "authentication", "authenticate", "sign in", "login", "credential", "publickey", "permission denied"):
		return "authentication unavailable", 10 * time.Minute
	case containsAny(low, "rate limit", "rate-limit", "too many requests", "quota", "429"):
		return "quota or rate limit", 5 * time.Minute
	case containsAny(low, "sandbox", "tool calls are denied", "requires approval", "approval required", "interactive approval", "permission mode"):
		return "worker tool permissions unavailable", 2 * time.Minute
	case containsAny(low, "unknown option", "unknown flag", "not found", "no such file or directory"):
		return "adapter or CLI mismatch", 5 * time.Minute
	case containsAny(low, "deadline exceeded", "timed out", "timeout"):
		return "worker timed out", 2 * time.Minute
	default:
		return "worker temporarily unavailable", time.Minute
	}
}

func backoffCooldown(base time.Duration, failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	shift := failures - 1
	if shift > 3 {
		shift = 3
	}
	d := base * time.Duration(1<<shift)
	if d > 30*time.Minute {
		return 30 * time.Minute
	}
	return d
}

func activeProviderHealth(st providerHealthState, now time.Time) map[string]ProviderHealth {
	out := map[string]ProviderHealth{}
	for _, entry := range st.Entries {
		if strings.TrimSpace(entry.ProviderID) != "" && entry.CooldownUntil.After(now) {
			out[entry.ProviderID] = entry
		}
	}
	return out
}

func compactCooldown(d time.Duration) string {
	if d >= time.Minute {
		minutes := int(d.Round(time.Minute) / time.Minute)
		if minutes < 1 {
			minutes = 1
		}
		return fmt.Sprintf("%dm", minutes)
	}
	seconds := int(d.Round(time.Second) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%ds", seconds)
}

func containsAny(s string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(s, value) {
			return true
		}
	}
	return false
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func loadProviderHealthState() (providerHealthState, error) {
	path, err := ProviderHealthStatePath()
	if err != nil {
		return providerHealthState{}, err
	}
	info, statErr := os.Stat(path)
	if errors.Is(statErr, os.ErrNotExist) {
		return providerHealthState{Version: providerHealthStateVersion}, nil
	}
	if statErr != nil {
		return providerHealthState{}, statErr
	}
	if info.Size() > maxProviderHealthBytes {
		return providerHealthState{}, errors.New("provider health cache is unexpectedly large")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return providerHealthState{}, err
	}
	var st providerHealthState
	if err := json.Unmarshal(b, &st); err != nil {
		return providerHealthState{}, err
	}
	if st.Version == 0 {
		st.Version = providerHealthStateVersion
	}
	return st, nil
}

func mutateProviderHealthState(fn func(*providerHealthState)) error {
	release, err := lockProviderHealthWrite()
	if err != nil {
		return err
	}
	defer release()
	st, err := loadProviderHealthState()
	if err != nil {
		return err
	}
	fn(&st)
	st.Version = providerHealthStateVersion
	return saveProviderHealthState(st)
}

func saveProviderHealthState(st providerHealthState) error {
	path, err := ProviderHealthStatePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > maxProviderHealthBytes {
		return errors.New("provider health cache exceeds its local size limit")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".provider-health-*.tmp")
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

func lockProviderHealthWrite() (func(), error) {
	providerHealthMu.Lock()
	path, err := ProviderHealthStatePath()
	if err != nil {
		providerHealthMu.Unlock()
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(3 * time.Second)
	for {
		f, openErr := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() {
				_ = os.Remove(lockPath)
				providerHealthMu.Unlock()
			}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			providerHealthMu.Unlock()
			return nil, openErr
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 2*time.Minute {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			providerHealthMu.Unlock()
			return nil, errors.New("timed out waiting for Workbench provider-health lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
