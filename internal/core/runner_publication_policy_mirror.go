package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// SaveRunnerProjectPublicationPolicyMirror records the operator's runner policy
// locally for fast Settings display and verified PR-link presentation. It never
// grants local Git publication authority: runner:// projects have no desktop
// worktree, and the actual execution policy must already have been accepted by
// the configured Workbench Runner before this mirror is written.
func SaveRunnerProjectPublicationPolicyMirror(policy PublicationPolicy) (PublicationPolicy, error) {
	name, ok := RunnerProjectName(policy.Project)
	if !ok {
		return PublicationPolicy{}, errors.New("runner publication-policy mirror requires a runner:// project")
	}
	project, err := RunnerProjectReference(name)
	if err != nil {
		return PublicationPolicy{}, err
	}
	policy.Project = project
	policy.RemoteURL = strings.TrimSpace(policy.RemoteURL)
	switch policy.Mode {
	case PublicationPrepare:
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
		if publicationPolicyKeysEqual(st.Policies[i].Project, project) {
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
	st.Aliases[publicationPolicyLookupIdentity(project)] = project
	if err := savePublicationPolicyState(st); err != nil {
		return PublicationPolicy{}, err
	}
	return policy, nil
}
