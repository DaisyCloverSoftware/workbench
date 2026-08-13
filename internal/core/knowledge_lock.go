package core

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// knowledgeMu protects in-process access. Writes also take a tiny lock-file
// lease because the desktop, MCP server and runner can be separate Workbench
// processes sharing the same knowledge.json file.
var knowledgeMu sync.RWMutex

const (
	knowledgeLockTimeout = 5 * time.Second
	knowledgeLockStale   = 2 * time.Minute
)

func lockKnowledgeWrite() (func(), error) {
	knowledgeMu.Lock()
	releaseFile, err := acquireKnowledgeFileLock()
	if err != nil {
		knowledgeMu.Unlock()
		return nil, err
	}
	return func() {
		releaseFile()
		knowledgeMu.Unlock()
	}, nil
}

func acquireKnowledgeFileLock() (func(), error) {
	path, err := KnowledgeStatePath()
	if err != nil {
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(knowledgeLockTimeout)
	for {
		f, openErr := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(openErr, os.ErrExist) {
			return nil, openErr
		}
		if st, statErr := os.Stat(lockPath); statErr == nil && time.Since(st.ModTime()) > knowledgeLockStale {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return nil, errors.New("timed out waiting for Workbench knowledge-store lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
