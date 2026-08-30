package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRUMIsolatedDevDeployerIsExactHeadAndLiveFailClosed(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	path := filepath.Join(root, "scripts", "ops", "rum-isolated-dev-deployer.sh")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read isolated DEV deploy wrapper: %v", err)
	}
	script := string(content)

	required := []string{
		`REPOSITORY="DaisyCloverSoftware/rum"`,
		`CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"`,
		`CANDIDATE_PR="153"`,
		`EXPECTED_CI_NAME="CI"`,
		`DEPLOY_SCRIPT="scripts/ops/deploy-rum-dev.sh"`,
		`requested SHA is not the exact current RUM owner-candidate branch head`,
		`no successful exact-head ${EXPECTED_CI_NAME} workflow run exists`,
		`RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate`,
		`git checkout --detach "$CANDIDATE_SHA"`,
		`NAMESPACE="rum-dev-isolated"`,
		`DEV_HOST="dev-rum.daisycloversoftware.uk"`,
		`PUBLIC_HOST="rateurmate.online"`,
		`RUM_LIVE_NAMESPACE_MUTATED=NO`,
		`RATE_ANYTHING_AFFECTED=NO`,
		`bash "$DEPLOY_SCRIPT" "$TAG"`,
	}
	for _, marker := range required {
		if !strings.Contains(script, marker) {
			t.Fatalf("isolated DEV deploy wrapper missing required marker %q", marker)
		}
	}

	forbidden := []string{
		"helm upgrade --install rum-dev",
		"kubectl -n rum-dev",
		"rate-anything-preview",
		"apps/rate-anything",
	}
	for _, marker := range forbidden {
		if strings.Contains(script, marker) {
			t.Fatalf("isolated DEV deploy wrapper contains forbidden marker %q", marker)
		}
	}
}
