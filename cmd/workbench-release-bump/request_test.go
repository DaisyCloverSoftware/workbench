package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReleaseRequestRejectsTrailingOrUnknownData(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"trailing-object": `{"version":"0.9.6","date":"2026-08-17","notes":["ok"]} {"extra":true}`,
		"trailing-garbage": `{"version":"0.9.6","date":"2026-08-17","notes":["ok"]} garbage`,
		"unknown-field": `{"version":"0.9.6","date":"2026-08-17","notes":["ok"],"surprise":true}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".json")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadReleaseRequest(path); err == nil {
				t.Fatalf("expected malformed request %q to be rejected", content)
			}
		})
	}
}
