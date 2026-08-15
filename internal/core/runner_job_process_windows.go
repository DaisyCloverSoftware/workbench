//go:build windows

package core

import "errors"

func spawnDetachedRunnerJob(jobID string) (int, error) {
	return 0, errors.New("durable cluster runner jobs require a Unix-like runner host")
}

func runnerJobProcessAlive(pid int) bool {
	return false
}

func terminateRunnerJobProcess(pid int) error {
	return nil
}
