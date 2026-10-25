package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRUMCandidateImagePublisherIsExactHeadAndRegistryOnly(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	path := filepath.Join(root, "scripts", "ops", "rum-candidate-image-publisher.sh")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read publisher: %v", err)
	}
	script := string(content)

	required := []string{
		`REPOSITORY="DaisyCloverSoftware/rum"`,
		`CANDIDATE_BRANCH="sprint-0-rum-owner-rating-flow-20260823"`,
		`CANDIDATE_PR="153"`,
		`requested SHA is not the exact current RUM owner-candidate branch head`,
		`no successful exact-head ${EXPECTED_CI_NAME} workflow run exists`,
		`RUM PR #${CANDIDATE_PR} is not the expected open, draft, unmerged exact candidate`,
		`ghcr.io/daisycloversoftware/rum-web:${TAG}`,
		`ghcr.io/daisycloversoftware/rum-api:${TAG}`,
		`VITE_DEMO_MODE=false`,
		`VITE_ENTITY_BRIDGE=true`,
		`VITE_UNIVERSAL_ENTITIES=false`,
		`org.opencontainers.image.revision=${CANDIDATE_SHA}`,
		`RUM_WEB_DIGEST=`,
		`RUM_API_DIGEST=`,
		`RATE_ANYTHING_AFFECTED=NO`,
		`LIVE_RUNTIME_AFFECTED=NO`,
	}
	for _, marker := range required {
		if !strings.Contains(script, marker) {
			t.Fatalf("publisher missing required marker %q", marker)
		}
	}

	forbidden := []string{
		"kubectl",
		"helm ",
		"rateurmate.online",
		"rum-rate-anything",
		"apps/rate-anything",
	}
	for _, marker := range forbidden {
		if strings.Contains(script, marker) {
			t.Fatalf("publisher contains forbidden marker %q", marker)
		}
	}
}
