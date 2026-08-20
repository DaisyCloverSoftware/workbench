//go:build windows

package core

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"
)

func runBlenderSmokeRender(ctx context.Context, executable, jobID string) (string, error) {
	dir, outputPrefix, expectedFile, err := blenderSmokeRenderPaths(jobID)
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", errors.New("Workbench could not prepare the Blender smoke render directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", errors.New("Workbench could not prepare the Blender smoke render directory")
	}
	name, args, err := blenderSmokeRenderInvocation(executable, outputPrefix)
	if err != nil {
		return "", err
	}

	renderCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(renderCtx, name, args...)
	configureChildProcess(cmd, false)
	// A smoke render is verified from the produced PNG, not Blender's console
	// chatter. Discarding both streams prevents diagnostics, paths or plug-in
	// messages from crossing the host boundary and avoids output-volume failures.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if renderCtx.Err() != nil {
			return "", errors.New("Blender smoke render timed out")
		}
		return "", errors.New("Blender smoke render failed")
	}
	return verifyBlenderSmokeRender(expectedFile)
}
