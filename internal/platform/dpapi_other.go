//go:build !windows

package platform

import "errors"

func ProtectString(value string) (string, error) {
	return "", errors.New("encrypted vault requires Windows DPAPI in this build")
}
func UnprotectString(value string) (string, error) {
	return "", errors.New("encrypted vault requires Windows DPAPI in this build")
}
