//go:build windows

package main

import "github.com/DaisyCloverSoftware/workbench/internal/desktop"

const appVersion = "0.7.0"

func main() {
	processOwnershipConfirmed := workbenchSingleInstanceHandle != 0
	if err := desktop.RunOwned(appVersion, processOwnershipConfirmed); err != nil {
		desktop.ShowError("Workbench could not start", err)
	}
}
