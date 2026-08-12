package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreateTunnelTokenIsStableAndBearerBacked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth-value")
	first, err := loadOrCreateTunnelToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(first) == "" {
		t.Fatal("generated token is empty")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stored := strings.TrimSpace(string(b))
	if stored != "Bearer "+first {
		t.Fatalf("stored auth value=%q, want Bearer <token>", stored)
	}
	second, err := loadOrCreateTunnelToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("token changed across restart")
	}
}
