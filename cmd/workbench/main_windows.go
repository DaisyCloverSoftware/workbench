//go:build windows

package main

import (
	"github.com/DaisyCloverSoftware/workbench/internal/core"
	"github.com/DaisyCloverSoftware/workbench/internal/desktop"
)

func main() {
	if err := desktop.Run(core.Version); err != nil {
		desktop.ShowError("Workbench could not start", err)
	}
}
