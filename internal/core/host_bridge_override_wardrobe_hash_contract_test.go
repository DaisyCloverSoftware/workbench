package core

import (
	"os"
	"strings"
	"testing"
)

func TestOverrideRinWardrobeInventoryUsesBoundedChangeDetectingHash(t *testing.T) {
	source, err := os.ReadFile("host_bridge_override_wardrobe_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"FileSHA256(match.absolute, overrideRinWardrobeMaxHashBytesPerFile)",
		"hashedSize != match.bytes",
		"hashedBytes += hashedSize",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("wardrobe scanner is missing bounded hash contract %q", required)
		}
	}
	if strings.Contains(text, "sha256RegularFile(match.absolute)") {
		t.Fatal("wardrobe scanner regressed to an unbounded candidate hash helper")
	}
}
