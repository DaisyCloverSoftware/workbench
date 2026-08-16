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
// helpers. SavePublicationPolicy retains the strict Git-root validation and
// records any lexical path alias presented at save time for later fast lookup.
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

	// Windows may hand Workbench an 8.3 spelling (for example RUNNER~1) while
	// Git/root validation expands it to the long directory spelling. The reader
	// must not touch the filesystem merely to rediscover that identity, so the
	// save path persists the original lexical key as an alias to the canonical
	// policy project.
	if canonical, ok := st.Aliases[publicationPolicyLookupIdentity(key)]; ok {
		for _, policy := range st.Policies {
			stored, keyErr := publicationPolicyReadKey(policy.Project)
			if keyErr == nil && publicationPolicyKeysEqual(stored, canonical) {
				return policy, true, nil
			}
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

func publicationPolicyLookupIdentity(key string) string {
	key = filepath.Clean(strings.TrimSpace(key))
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}

func publicationPolicyKeysEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
