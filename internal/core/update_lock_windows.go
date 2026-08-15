//go:build windows

package core

import "errors"

type UpdateLock struct{}

func AcquireUpdateLock() (*UpdateLock, error) {
	return nil, errors.New("cluster self-update apply is supported on Linux hosts only")
}

func (l *UpdateLock) Close() error { return nil }
