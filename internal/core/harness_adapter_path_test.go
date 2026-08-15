package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateHarnessAdapterPathRejectsShellOrNonExecutableFiles(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		batch := filepath.Join(dir, "adapter.cmd")
		if err := os.WriteFile(batch, []byte("@echo off\r\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateHarnessAdapterPath(batch); err == nil {
			t.Fatal("Windows batch file was accepted as a structured adapter executable")
		}
		native := filepath.Join(dir, "adapter.exe")
		if err := os.WriteFile(native, []byte("native fixture"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateHarnessAdapterPath(native); err != nil {
			t.Fatalf("native Windows executable path was rejected: %v", err)
		}
		return
	}

	adapter := filepath.Join(dir, "adapter")
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateHarnessAdapterPath(adapter); err == nil {
		t.Fatal("non-executable Unix adapter file was accepted")
	}
	if err := os.Chmod(adapter, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateHarnessAdapterPath(adapter); err != nil {
		t.Fatalf("executable Unix adapter was rejected: %v", err)
	}
}
