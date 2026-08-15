package core

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
)

// PublicationPolicyForKnownProject reads an operator policy for a project that
// has already been registered and canonicalised by Workbench. Unlike the
// mutation path, this lookup deliberately performs no Git command and no
// filesystem probing, so opening the desktop Settings page cannot be held up by
// repository discovery, a slow remote filesystem, Git hooks, or credential
// helpers. SavePublicationPolicy retains the strict Git-root validation.
func PublicationPolicyForKnownProject(project string) (PublicationPolicy, bool, error) {
	key, err := publicationPolicyReadKey(project)
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
		stored, keyErr := publicationPolicyReadKey(policy.Project)
		if keyErr == nil && publicationPolicyKeysEqual(stored, key) {
			return policy, true, nil
		}
	}
	return PublicationPolicy{}, false, nil
}

func publicationPolicyReadKey(project string) (string, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return "", errors.New("publication project is empty")
	}
	abs, err := filepath.Abs(project)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func publicationPolicyKeysEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
