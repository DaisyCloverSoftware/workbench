package core

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// githubSlugFromRemoteForEnsure resolves the repository identity of a GitHub
// origin without ever returning embedded userinfo. Ordinary credential-free
// forms use the stricter shared parser. The HTTPS fallback exists only so
// ensure_github_project can recognise a legacy same-repository origin that has
// userinfo embedded in the URL and replace it with the fixed SSH form.
func githubSlugFromRemoteForEnsure(remote string) (string, bool) {
	if slug, ok := githubSlugFromRemote(remote); ok {
		return slug, true
	}

	u, err := url.Parse(strings.TrimSpace(remote))
	if err != nil || !strings.EqualFold(u.Scheme, "https") || !strings.EqualFold(u.Hostname(), "github.com") || u.Port() != "" || u.User == nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return "", false
	}
	slug := strings.Trim(strings.TrimSpace(u.Path), "/")
	slug = strings.TrimSuffix(slug, ".git")
	owner, name, err := validateGitHubRepositorySlug(slug)
	if err != nil {
		return "", false
	}
	return strings.ToLower(owner + "/" + name), true
}

// ensureRunnerGitHubProjectOriginReady keeps project discovery non-destructive
// while repairing one narrow unsafe legacy condition: a checkout whose origin
// proves it is the requested github.com owner/name but is not acceptable for
// exact-commit operations because the URL contains credentials (or otherwise
// uses a non-approved GitHub transport). The replacement is the fixed-host SSH
// URL used by Workbench's own clone fallback. The prior URL is never returned or
// logged, and verification rolls the change back if the fixed origin cannot be
// established exactly.
func ensureRunnerGitHubProjectOriginReady(ctx context.Context, project RunnerProjectInfo, owner, name string) error {
	root, err := ResolveRunnerProject(project.Ref)
	if err != nil {
		return errors.New("runner GitHub project could not be resolved for origin readiness")
	}
	current, err := runGitLimited(ctx, root, 4096, "remote", "get-url", "origin")
	if err != nil {
		return errors.New("runner GitHub project origin could not be inspected")
	}
	current = strings.TrimSpace(current)
	wantSlug := strings.ToLower(owner + "/" + name)
	slug, ok := githubSlugFromRemoteForEnsure(current)
	if !ok || !strings.EqualFold(slug, wantSlug) {
		return errors.New("runner GitHub project origin does not match the requested repository")
	}
	if isApprovedOperationsOrigin(current) {
		return nil
	}

	desired := "git@github.com:" + owner + "/" + name + ".git"
	if _, err := runGitLimited(ctx, root, 4096, "remote", "set-url", "origin", desired); err != nil {
		return errors.New("runner GitHub project origin could not be normalised")
	}
	verified, verifyErr := runGitLimited(ctx, root, 4096, "remote", "get-url", "origin")
	verified = strings.TrimSpace(verified)
	verifiedSlug, verifiedOK := githubSlugFromRemote(verified)
	if verifyErr != nil || !isApprovedOperationsOrigin(verified) || !verifiedOK || !strings.EqualFold(verifiedSlug, wantSlug) {
		_, _ = runGitLimited(ctx, root, 4096, "remote", "set-url", "origin", current)
		return errors.New("runner GitHub project origin normalisation did not verify")
	}
	return nil
}
