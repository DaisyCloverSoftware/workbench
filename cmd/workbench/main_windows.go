//go:build windows

package main

import (
	"errors"

	"github.com/DaisyCloverSoftware/workbench/internal/desktop"
)

const appVersion = "0.9.13"

func main() {
	if workbenchSingleInstanceHandle == 0 {
		desktop.ShowError("Workbench could not start", errors.New("Workbench could not acquire its per-user desktop ownership mutex; no durable state or coding work was started"))
		return
	}
	if err := desktop.RunOwned(appVersion, true); err != nil {
		desktop.ShowError("Workbench could not start", err)
	}
}
