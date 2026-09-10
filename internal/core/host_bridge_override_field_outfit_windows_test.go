//go:build windows

package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOverrideRinFieldOutfitCopyExactPreservesSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.blend")
	destination := filepath.Join(dir, "copy.blend")
	data := []byte("sealed field outfit fixture\n")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, size, err := FileSHA256(source, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	pin := overrideRinFieldOutfitPinnedFile{Role: "fixture", Path: "SourceAssets/RinWardrobe/source.blend", Bytes: size, SHA256: digest}
	if err := overrideRinFieldOutfitCopyExact(source, destination, pin); err != nil {
		t.Fatal(err)
	}
	sourceAfter, sizeAfter, err := FileSHA256(source, int64(len(data)))
	if err != nil || sourceAfter != digest || sizeAfter != size {
		t.Fatal("source changed while staging fixture")
	}
	if err := overrideRinFieldOutfitVerifyFile(destination, pin); err != nil {
		t.Fatal(err)
	}
}
