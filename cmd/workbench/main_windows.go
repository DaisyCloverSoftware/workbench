//go:build windows

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/desktop"
)

const appVersion = "0.9.44"

func main() {
	if workbenchSingleInstanceHandle == 0 {
		desktop.ShowError("Workbench could not start", errors.New("Workbench could not acquire its per-user desktop ownership mutex; no durable state or coding work was started"))
		return
	}
	// The updater is a separate executable and therefore cannot reliably replace
	// itself while it is running. Refresh it silently from the same verified
	// release channel after Workbench owns the desktop. This also repairs older
	// installations whose Workbench.exe updated successfully while their updater
	// binary remained on an earlier release.
	go refreshVerifiedUpdater()
	if err := desktop.RunOwned(appVersion, true); err != nil {
		desktop.ShowError("Workbench could not start", err)
	}
}

func refreshVerifiedUpdater() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	_, _ = core.RefreshVerifiedWindowsUpdater(ctx, filepath.Dir(exe))
}
