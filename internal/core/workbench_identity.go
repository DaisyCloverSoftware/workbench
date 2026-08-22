package core

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const maxWorkbenchIdentityBytes = int64(128 << 20)

func workbenchExecutableIdentity(path, version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" || len(version) > 64 || LooksSecret(version) {
		return "", errors.New("Workbench version is invalid")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", errors.New("Workbench executable could not be inspected")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("Workbench executable is not a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxWorkbenchIdentityBytes {
		return "", errors.New("Workbench executable size is outside the allowed range")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", errors.New("Workbench executable could not be opened for identity verification")
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxWorkbenchIdentityBytes+1))
	if err != nil || n != info.Size() || n > maxWorkbenchIdentityBytes {
		return "", errors.New("Workbench executable changed while its identity was being verified")
	}
	after, err := f.Stat()
	if err != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return "", errors.New("Workbench executable changed while its identity was being verified")
	}
	return fmt.Sprintf("Workbench %s sha256=%x bytes=%d", version, h.Sum(nil), n), nil
}

func runningWorkbenchExecutableIdentity() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", errors.New("running Workbench executable could not be identified")
	}
	return workbenchExecutableIdentity(executable, Version)
}
