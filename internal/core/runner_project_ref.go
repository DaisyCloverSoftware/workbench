package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const RunnerProjectPrefix = "runner://"

func validateRunnerProjectName(name string) (string, error) {
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
	return name, nil
}

// RunnerProjectReference returns the backwards-compatible unscoped reference
// for a repository name. Unscoped references remain stable when a repository
// name is unique across all authorised runner roots.
func RunnerProjectReference(name string) (string, error) {
	name, err := validateRunnerProjectName(name)
	if err != nil {
		return "", err
	}
	return RunnerProjectPrefix + name, nil
}

// RunnerScopedProjectReference disambiguates the same repository directory name
// appearing under more than one authorised runner root without revealing host
// filesystem paths. Root numbers are the stable operator-configured root order.
func RunnerScopedProjectReference(rootNumber int, name string) (string, error) {
	if rootNumber <= 0 || rootNumber > 9999 {
		return "", errors.New("runner project root number is invalid")
	}
	name, err := validateRunnerProjectName(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%sr%d/%s", RunnerProjectPrefix, rootNumber, name), nil
}

// RunnerProjectLocator parses both legacy runner://name and scoped
// runner://rN/name references. rootNumber is zero for a legacy unscoped ref.
func RunnerProjectLocator(ref string) (rootNumber int, name string, ok bool) {
	ref = strings.TrimSpace(ref)
	if len(ref) < len(RunnerProjectPrefix) || !strings.EqualFold(ref[:len(RunnerProjectPrefix)], RunnerProjectPrefix) {
		return 0, "", false
	}
	rest := strings.TrimSpace(ref[len(RunnerProjectPrefix):])
	if rest == "" {
		return 0, "", false
	}
	if !strings.Contains(rest, "/") {
		name, err := validateRunnerProjectName(rest)
		return 0, name, err == nil
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || len(parts[0]) < 2 || (parts[0][0] != 'r' && parts[0][0] != 'R') {
		return 0, "", false
	}
	n, err := strconv.Atoi(parts[0][1:])
	if err != nil || n <= 0 || n > 9999 {
		return 0, "", false
	}
	name, err = validateRunnerProjectName(parts[1])
	if err != nil {
		return 0, "", false
	}
	return n, name, true
}

func NormalizeRunnerProjectReference(ref string) (string, bool) {
	rootNumber, name, ok := RunnerProjectLocator(ref)
	if !ok {
		return "", false
	}
	var (
		normalized string
		err        error
	)
	if rootNumber == 0 {
		normalized, err = RunnerProjectReference(name)
	} else {
		normalized, err = RunnerScopedProjectReference(rootNumber, name)
	}
	return normalized, err == nil
}

func RunnerProjectName(ref string) (string, bool) {
	_, name, ok := RunnerProjectLocator(ref)
	return name, ok
}

func IsRunnerProjectReference(ref string) bool {
	_, _, ok := RunnerProjectLocator(ref)
	return ok
}
