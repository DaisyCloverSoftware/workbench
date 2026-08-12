package core

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPublicSourcePrivacy is intentionally conservative: the public repository
// must stay environment-agnostic. Use documentation-only example ranges and
// placeholders instead of copying private deployment values into source.
func TestPublicSourcePrivacy(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	rules := []struct {
		name string
		re   *regexp.Regexp
	}{
		{"unix home path", regexp.MustCompile(`/` + `home/` + `[A-Za-z0-9._-]+/`)},
		{"macOS home path", regexp.MustCompile(`/` + `Users/` + `[A-Za-z0-9._-]+/`)},
		{"Windows user profile path", regexp.MustCompile(`(?i)[A-Z]:\\` + `Users\\` + `[A-Za-z0-9._-]+\\`)},
		{"tailnet host name", regexp.MustCompile(`(?i)\b[a-z0-9-]+\.[a-z0-9-]+\.` + `ts\.net\b`)},
		{"Tailscale CGNAT address", regexp.MustCompile(`\b100\.(?:6[4-9]|[7-9][0-9]|1[01][0-9]|12[0-7])\.[0-9]{1,3}\.[0-9]{1,3}\b`)},
		{"RFC1918 10/8 address", regexp.MustCompile(`\b10\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\b`)},
		{"RFC1918 172.16/12 address", regexp.MustCompile(`\b172\.(?:1[6-9]|2[0-9]|3[01])\.[0-9]{1,3}\.[0-9]{1,3}\b`)},
		{"RFC1918 192.168/16 address", regexp.MustCompile(`\b192\.168\.[0-9]{1,3}\.[0-9]{1,3}\b`)},
		{"shell user-at-host prompt", regexp.MustCompile(`(?i)\b[a-z][a-z0-9._-]*@[a-z0-9][a-z0-9.-]*:[~/]`)},
	}

	skipDirs := map[string]bool{
		".git": true, ".idea": true, ".vscode": true,
		"node_modules": true, "vendor": true, "dist": true, "build": true,
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > 2<<20 {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(b, 0) >= 0 {
			return nil
		}
		text := string(b)
		rel, _ := filepath.Rel(root, path)
		for _, rule := range rules {
			for _, match := range rule.re.FindAllString(text, -1) {
				if rule.name == "tailnet host name" && strings.EqualFold(match, "your-agent-host.tailnet.ts.net") {
					continue
				}
				t.Errorf("public source privacy rule %q matched %s", rule.name, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
