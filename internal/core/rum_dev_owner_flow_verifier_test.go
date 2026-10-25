package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRUMDevOwnerFlowVerifierUsesExactCIValidatedCandidate(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	path := filepath.Join(root, "scripts", "ops", "rum-dev-owner-flow-verifier.sh")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read verifier wrapper: %v", err)
	}
	script := string(content)

	required := []string{
		`REPOSITORY="DaisyCloverSoftware/rum"`,
		`CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"`,
		`CANDIDATE_PR="153"`,
		`VERIFIER_PATH="scripts/ops/verify-rum-dev-owner-rating-flow.sh"`,
		`requested SHA is not the exact current RUM owner-candidate branch head`,
		`no successful exact-head ${EXPECTED_CI_NAME} workflow run exists`,
		`RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate`,
		`git checkout --detach "$CANDIDATE_SHA"`,
		`command -v podman`,
		`ln -s "$(command -v podman)" "$runtime_bin/docker"`,
		`RUM_OWNER_FLOW_CONTAINER_RUNTIME=podman`,
		`neither Docker nor Podman is available`,
		`unset TOKEN GH_TOKEN GHCR_TOKEN`,
		`RUM_OWNER_FLOW_EXECUTED_VERIFIER=%s`,
		`bash "$compat_verifier"`,
	}
	for _, marker := range required {
		if !strings.Contains(script, marker) {
			t.Fatalf("verifier wrapper missing required marker %q", marker)
		}
	}

	forbidden := []string{
		"rate-anything-preview",
		"apps/rate-anything",
	}
	for _, marker := range forbidden {
		if strings.Contains(script, marker) {
			t.Fatalf("verifier wrapper contains forbidden marker %q", marker)
		}
	}
}
