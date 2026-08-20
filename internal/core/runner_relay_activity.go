package core

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxRelayActivityArchive = 8 << 20
	runnerChatSessionWindow  = 4 * time.Hour
)

type RunnerChatActivityInfo struct {
	ID          string    `json:"id"`
	ProjectRef  string    `json:"project_ref"`
	Action      string    `json:"action"`
	State       string    `json:"state"`
	UpdatedAt   time.Time `json:"updated_at"`
	Active      bool      `json:"active"`
	ActiveKnown bool      `json:"active_known"`
}

type relayControlActivity struct {
	ID      string `json:"id"`
	Action  string `json:"action"`
	Project string `json:"project,omitempty"`
}

type relayControlActivityResult struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type relayAutonomousActivity struct {
	ID      string `json:"id"`
	Project string `json:"project"`
}

type relayAutonomousActivityResult struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

func ReadRunnerChatActivity(limit int) ([]RunnerChatActivityInfo, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	repo, err := runnerRelayRepoDir()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", "origin/main", "--", "relay/control", "relay/control-outbox", "relay/inbox", "relay/outbox")
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.New("private Workbench relay activity is unavailable")
	}
	if len(out) > maxRelayActivityArchive {
		return nil, errors.New("private Workbench relay activity exceeded bounds")
	}
	items, err := parseRunnerChatActivity(out, limit)
	if err != nil {
		return nil, err
	}
	// Activity timestamps originate on this runner. Decide the bounded ChatGPT
	// session lease here too, on the same clock, instead of asking the Windows
	// desktop to compare a runner timestamp with a potentially skewed local clock.
	now := time.Now().UTC()
	for i := range items {
		items[i].ActiveKnown = true
		items[i].Active = runnerChatActivityIsActive(items[i], now)
	}
	return items, nil
}

func runnerChatActivityIsActive(event RunnerChatActivityInfo, now time.Time) bool {
	state := strings.ToLower(strings.TrimSpace(event.State))
	if state == "running" || state == "waiting" || state == "needs_attention" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(event.Action), "delegate_task") {
		return false
	}
	if event.UpdatedAt.IsZero() {
		return false
	}
	return !event.UpdatedAt.Before(now.Add(-runnerChatSessionWindow))
}

func runnerRelayRepoDir() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("WORKBENCH_RELAY_REPO_DIR")); configured != "" {
		return validateRunnerRelayRepo(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return validateRunnerRelayRepo(filepath.Join(home, ".local", "share", "workbench", "relay-private"))
}

func validateRunnerRelayRepo(path string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return "", errors.New("private Workbench relay repository is unavailable")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("private Workbench relay repository is unavailable")
	}
	if stat, err := os.Stat(filepath.Join(path, ".git")); err != nil || !stat.IsDir() {
		return "", errors.New("private Workbench relay repository is unavailable")
	}
	return path, nil
}

func parseRunnerChatActivity(raw []byte, limit int) ([]RunnerChatActivityInfo, error) {
	controls := map[string]relayControlActivity{}
	controlResults := map[string]relayControlActivityResult{}
	autonomous := map[string]relayAutonomousActivity{}
	autonomousResults := map[string]relayAutonomousActivityResult{}

	tr := tar.NewReader(bytes.NewReader(raw))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("private Workbench relay activity archive is invalid")
		}
		if hdr.FileInfo().IsDir() || hdr.Size < 0 || hdr.Size > 256<<10 {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, 256<<10))
		if err != nil {
			continue
		}
		name := filepath.ToSlash(hdr.Name)
		switch {
		case strings.HasPrefix(name, "relay/control/") && strings.HasSuffix(name, ".json"):
			var v relayControlActivity
			if json.Unmarshal(b, &v) == nil && strings.TrimSpace(v.ID) != "" {
				controls[v.ID] = v
			}
		case strings.HasPrefix(name, "relay/control-outbox/") && strings.HasSuffix(name, ".json"):
			var v relayControlActivityResult
			if json.Unmarshal(b, &v) == nil && strings.TrimSpace(v.ID) != "" {
				controlResults[v.ID] = v
			}
		case strings.HasPrefix(name, "relay/inbox/") && strings.HasSuffix(name, ".json"):
			var v relayAutonomousActivity
			if json.Unmarshal(b, &v) == nil && strings.TrimSpace(v.ID) != "" {
				autonomous[v.ID] = v
			}
		case strings.HasPrefix(name, "relay/outbox/") && strings.HasSuffix(name, ".json"):
			var v relayAutonomousActivityResult
			if json.Unmarshal(b, &v) == nil && strings.TrimSpace(v.ID) != "" {
				autonomousResults[v.ID] = v
			}
		}
	}

	items := make([]RunnerChatActivityInfo, 0, len(controls)+len(autonomous))
	for id, result := range controlResults {
		request, ok := controls[id]
		if !ok {
			continue
		}
		project := strings.TrimSpace(request.Project)
		if project != "" && !IsRunnerProjectReference(project) {
			continue
		}
		updated, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(result.UpdatedAt))
		if err != nil {
			continue
		}
		action := strings.TrimSpace(result.Action)
		if action == "" {
			action = strings.TrimSpace(request.Action)
		}
		items = append(items, RunnerChatActivityInfo{ID: id, ProjectRef: project, Action: action, State: normalizeChatActivityState(result.Status), UpdatedAt: updated.UTC()})
	}
	for id, request := range controls {
		if _, done := controlResults[id]; done {
			continue
		}
		project := strings.TrimSpace(request.Project)
		if project != "" && !IsRunnerProjectReference(project) {
			continue
		}
		items = append(items, RunnerChatActivityInfo{ID: id, ProjectRef: project, Action: strings.TrimSpace(request.Action), State: "running"})
	}
	for id, result := range autonomousResults {
		request, ok := autonomous[id]
		if !ok || !IsRunnerProjectReference(strings.TrimSpace(request.Project)) {
			continue
		}
		updated, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(result.UpdatedAt))
		if err != nil {
			continue
		}
		items = append(items, RunnerChatActivityInfo{ID: id, ProjectRef: strings.TrimSpace(request.Project), Action: "delegate_task", State: normalizeChatActivityState(result.Status), UpdatedAt: updated.UTC()})
	}
	for id, request := range autonomous {
		if _, done := autonomousResults[id]; done || !IsRunnerProjectReference(strings.TrimSpace(request.Project)) {
			continue
		}
		items = append(items, RunnerChatActivityInfo{ID: id, ProjectRef: strings.TrimSpace(request.Project), Action: "delegate_task", State: "running"})
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func normalizeChatActivityState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "completed", "success":
		return "completed"
	case "failed", "cancelled":
		return "failed"
	case "needs_attention":
		return "needs_attention"
	case "waiting_retry", "waiting_dependency", "waiting":
		return "waiting"
	case "queued", "routing", "running":
		return "running"
	default:
		return "completed"
	}
}
