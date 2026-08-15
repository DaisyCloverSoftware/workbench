//go:build !windows

package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type UpdateLock struct {
	file *os.File
}

func AcquireUpdateLock() (*UpdateLock, error) {
	base, err := ScratchBaseDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, "update.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errors.New("another Workbench update is already running")
		}
		return nil, fmt.Errorf("lock Workbench update: %w", err)
	}
	return &UpdateLock{file: f}, nil
}

func (l *UpdateLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
