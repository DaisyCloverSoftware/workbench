package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var workflowUsePattern = regexp.MustCompile(`(?m)^\s*-\s+uses:\s+([^\s#]+)`)
var immutableActionRefPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func TestGitHubWorkflowsPinThirdPartyActionsToImmutableCommits(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	paths, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no GitHub workflows found")
	}

	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, match := range workflowUsePattern.FindAllStringSubmatch(string(b), -1) {
			use := match[1]
			if strings.HasPrefix(use, "./") || strings.HasPrefix(use, "docker://") {
				continue
			}
			at := strings.LastIndexByte(use, '@')
			if at <= 0 || !immutableActionRefPattern.MatchString(use[at+1:]) {
				t.Errorf("%s contains mutable action reference %q; pin third-party actions to a full 40-character commit SHA", filepath.Base(path), use)
			}
		}
	}
}
