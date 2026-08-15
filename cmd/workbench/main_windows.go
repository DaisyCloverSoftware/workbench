//go:build windows

package main

import "github.com/DaisyCloverSoftware/workbench/internal/desktop"

const appVersion = "0.7.0"

func main() {
	if err := desktop.Run(appVersion); err != nil {
		desktop.ShowError("Workbench could not start", err)
	}
}
