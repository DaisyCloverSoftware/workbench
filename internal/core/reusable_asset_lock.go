package core

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	reusableAssetLockTimeout = 5 * time.Second
	reusableAssetLockStale   = 2 * time.Minute
)

func lockReusableAssetWrite() (func(), error) {
	reusableAssetMu.Lock()
	path, err := ReusableAssetStatePath()
	if err != nil {
		reusableAssetMu.Unlock()
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(reusableAssetLockTimeout)
	for {
		f, openErr := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() {
				_ = os.Remove(lockPath)
				reusableAssetMu.Unlock()
			}, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			reusableAssetMu.Unlock()
			return nil, openErr
		}
		if st, statErr := os.Stat(lockPath); statErr == nil && time.Since(st.ModTime()) > reusableAssetLockStale {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			reusableAssetMu.Unlock()
			return nil, errors.New("timed out waiting for Workbench reusable-asset lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
