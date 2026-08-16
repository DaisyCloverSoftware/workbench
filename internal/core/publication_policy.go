package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PublicationMode string

const (
	PublicationPrepare PublicationMode = "prepare"
	PublicationPublish PublicationMode = "publish"
)

type PublicationPolicy struct {
	Project   string          `json:"project"`
	Mode      PublicationMode `json:"mode"`
	RemoteURL string          `json:"remote_url,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type publicationPolicyState struct {
	Version  int                 `json:"version"`
	Policies []PublicationPolicy `json:"policies"`
	// Aliases map a lexical project spelling observed during an explicit save
	// to the canonical project stored in Policies. This lets the read-only UI
	// path resolve Windows 8.3/long-path aliases without touching Git or the
	// filesystem when Settings is opened later.
	Aliases map[string]string `json:"aliases,omitempty"`
}

const maxPublicationPolicyBytes = 1 << 20

var publicationPolicyMu sync.RWMutex

func PublicationPolicyStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "Workbench")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "publication-policy.json"), nil
}

// SavePublicationPolicy stores a local operator policy. Publication targets are
// deliberately kept outside task state, worker prompts, MCP workspace output,
// and the Git relay so coding workers cannot choose where source is published.
func SavePublicationPolicy(policy PublicationPolicy) (PublicationPolicy, error) {
	inputKey, err := publicationPolicyReadKey(policy.Project)
	if err != nil {
		return PublicationPolicy{}, err
	}
	project, err := canonicalPublicationProject(policy.Project)
	if err != nil {
		return PublicationPolicy{}, err
	}
	policy.Project = project
	policy.RemoteURL = strings.TrimSpace(policy.RemoteURL)
	switch policy.Mode {
	case PublicationPrepare:
		// Preparation is local-only. Ignore a stale target instead of carrying
		// unnecessary private repository metadata in a prepare-only policy.
		policy.RemoteURL = ""
	case PublicationPublish:
		if err := validatePublishRemote(policy.RemoteURL); err != nil {
			return PublicationPolicy{}, fmt.Errorf("invalid publication target: %w", err)
		}
	default:
		return PublicationPolicy{}, errors.New("publication mode must be prepare or publish")
	}
	policy.UpdatedAt = time.Now().UTC()

	release, err := lockPublicationPolicyWrite()
	if err != nil {
		return PublicationPolicy{}, err
	}
	defer release()
	st, err := loadPublicationPolicyState()
	if err != nil {
		return PublicationPolicy{}, err
	}
	updated := false
	for i := range st.Policies {
		if st.Policies[i].Project == policy.Project {
			st.Policies[i] = policy
			updated = true
			break
		}
	}
	if !updated {
		st.Policies = append(st.Policies, policy)
	}
	if st.Aliases == nil {
		st.Aliases = map[string]string{}
	}
	canonicalKey, keyErr := publicationPolicyReadKey(policy.Project)
	if keyErr != nil {
		return PublicationPolicy{}, keyErr
	}
	st.Aliases[publicationPolicyLookupIdentity(inputKey)] = canonicalKey
	st.Aliases[publicationPolicyLookupIdentity(canonicalKey)] = canonicalKey
	if err := savePublicationPolicyState(st); err != nil {
		return PublicationPolicy{}, err
	}
	return policy, nil
}

func PublicationPolicyFor(project string) (PublicationPolicy, bool, error) {
	project, err := canonicalPublicationProject(project)
	if err != nil {
		return PublicationPolicy{}, false, err
	}
	publicationPolicyMu.RLock()
	defer publicationPolicyMu.RUnlock()
	st, err := loadPublicationPolicyState()
	if err != nil {
		return PublicationPolicy{}, false, err
	}
	for _, policy := range st.Policies {
		if policy.Project == project {
			return policy, true, nil
		}
	}
	return PublicationPolicy{}, false, nil
}

func DeletePublicationPolicy(project string) error {
	project, err := canonicalPublicationProject(project)
	if err != nil {
		return err
	}
	release, err := lockPublicationPolicyWrite()
	if err != nil {
		return err
	}
	defer release()
	st, err := loadPublicationPolicyState()
	if err != nil {
		return err
	}
	out := st.Policies[:0]
	for _, policy := range st.Policies {
		if policy.Project != project {
			out = append(out, policy)
		}
	}
	st.Policies = out
	if len(st.Aliases) > 0 {
		for alias, canonical := range st.Aliases {
			if publicationPolicyKeysEqual(canonical, project) {
				delete(st.Aliases, alias)
			}
		}
		if len(st.Aliases) == 0 {
			st.Aliases = nil
		}
	}
	return savePublicationPolicyState(st)
}

func canonicalPublicationProject(project string) (string, error) {
	root, err := projectRoot(project)
	if err != nil {
		return "", err
	}
	// Policy mutation is an explicit operator action, but still keep the Git
	// validation bounded so a dead filesystem cannot pin the desktop forever.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	gitRoot, err := runGitLimited(ctx, root, 16<<10, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("publication policy requires a responsive Git repository root")
	}
	gitRoot = strings.TrimSpace(gitRoot)
	resolvedGitRoot, err := filepath.EvalSymlinks(gitRoot)
	if err != nil {
		return "", err
	}
	resolvedGitRoot, err = filepath.Abs(resolvedGitRoot)
	if err != nil {
		return "", err
	}
	if filepath.Clean(root) != filepath.Clean(resolvedGitRoot) {
		return "", errors.New("publication policy must be configured at the Git repository root")
	}
	return root, nil
}

func loadPublicationPolicyState() (publicationPolicyState, error) {
	path, err := PublicationPolicyStatePath()
	if err != nil {
		return publicationPolicyState{}, err
	}
	stInfo, statErr := os.Stat(path)
	if errors.Is(statErr, os.ErrNotExist) {
		return publicationPolicyState{Version: 1}, nil
	}
	if statErr != nil {
		return publicationPolicyState{}, statErr
	}
	if stInfo.Size() > maxPublicationPolicyBytes {
		return publicationPolicyState{}, errors.New("publication policy store is unexpectedly large")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return publicationPolicyState{}, err
	}
	var st publicationPolicyState
	if err := json.Unmarshal(b, &st); err != nil {
		return publicationPolicyState{}, err
	}
	if st.Version == 0 {
		st.Version = 1
	}
	return st, nil
}

func savePublicationPolicyState(st publicationPolicyState) error {
	path, err := PublicationPolicyStatePath()
	if err != nil {
		return err
	}
	st.Version = 1
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > maxPublicationPolicyBytes {
		return errors.New("publication policy store exceeds its local size limit")
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".publication-policy-*.tmp")
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

func lockPublicationPolicyWrite() (func(), error) {
	publicationPolicyMu.Lock()
	path, err := PublicationPolicyStatePath()
	if err != nil {
		publicationPolicyMu.Unlock()
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
				publicationPolicyMu.Unlock()
			}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			publicationPolicyMu.Unlock()
			return nil, openErr
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 2*time.Minute {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			publicationPolicyMu.Unlock()
			return nil, errors.New("timed out waiting for Workbench publication-policy lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
