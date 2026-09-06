//go:build !windows

package runnerdiag

import "errors"

func Inspect() (Report, error) {
	return newReport(), errors.New("Windows runner diagnostic requires Windows")
}
func InspectJSON() (string, error) {
	return "", errors.New("Windows runner diagnostic requires Windows")
}
