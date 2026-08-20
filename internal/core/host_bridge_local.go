package core

import (
	"errors"
	"path/filepath"
	"strings"
)

func blenderVersionInvocation(executable string) (string, []string, error) {
	executable = strings.TrimSpace(executable)
	if executable == "" {
		return "", nil, errors.New("Blender executable is empty")
	}
	if strings.ContainsAny(executable, "\r\n\x00") {
		return "", nil, errors.New("Blender executable contains invalid characters")
	}
	if !strings.EqualFold(filepath.Base(executable), "blender.exe") {
		return "", nil, errors.New("Blender executable must be blender.exe")
	}
	return executable, []string{"--version"}, nil
}

func blenderVersionFromCapturedOutput(stdout, stderr *boundedWorkerCapture) (string, error) {
	// Blender's authoritative version line is expected on stdout. Accept that
	// bounded, validated line even when unrelated diagnostics overflow stderr (or
	// later stdout), while still failing closed when no valid version was captured.
	if version, err := parseBlenderVersionOutput(stdout.String()); err == nil {
		return version, nil
	}
	if stdout.Truncated() || stderr.Truncated() {
		return "", errors.New("Blender version output exceeded Workbench limits before a valid version was captured")
	}

	combined := strings.TrimSpace(stdout.String())
	if se := strings.TrimSpace(stderr.String()); se != "" {
		if combined != "" {
			combined += "\n"
		}
		combined += se
	}
	return parseBlenderVersionOutput(combined)
}

func parseBlenderVersionOutput(output string) (string, error) {
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "Blender ") {
			return "", errors.New("Blender returned an unexpected version response")
		}
		if len(line) > 256 || LooksSecret(line) {
			return "", errors.New("Blender version response is invalid")
		}
		return line, nil
	}
	return "", errors.New("Blender returned no version")
}
