package core

import (
	"errors"
	"strings"
)

const RunnerProjectPrefix = "runner://"

func RunnerProjectReference(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("runner project name is empty")
	}
	if len(name) > 160 {
		return "", errors.New("runner project name is too long")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, "/\\:\x00\r\n\t") {
		return "", errors.New("runner project name must be one safe directory name")
	}
	return RunnerProjectPrefix + name, nil
}

func RunnerProjectName(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if len(ref) < len(RunnerProjectPrefix) || !strings.EqualFold(ref[:len(RunnerProjectPrefix)], RunnerProjectPrefix) {
		return "", false
	}
	name := strings.TrimSpace(ref[len(RunnerProjectPrefix):])
	if _, err := RunnerProjectReference(name); err != nil {
		return "", false
	}
	return name, true
}

func IsRunnerProjectReference(ref string) bool {
	_, ok := RunnerProjectName(ref)
	return ok
}
