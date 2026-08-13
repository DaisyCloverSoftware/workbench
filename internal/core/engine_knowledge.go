package core

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var knowledgeStores sync.Map

// KnowledgeStore returns the durable knowledge database paired with this
// engine's state store. Tests that use NewStoreAt therefore stay isolated from
// the real user's knowledge database.
func (e *Engine) KnowledgeStore() (*KnowledgeStore, error) {
	path := filepath.Join(filepath.Dir(e.store.Path()), "knowledge.json")
	if v, ok := knowledgeStores.Load(path); ok {
		return v.(*KnowledgeStore), nil
	}
	store, err := NewKnowledgeStoreAt(path)
	if err != nil {
		return nil, err
	}
	actual, _ := knowledgeStores.LoadOrStore(path, store)
	return actual.(*KnowledgeStore), nil
}

func (e *Engine) Remember(project string, scope MemoryScope, kind MemoryKind, title, summary, content string, tags []string, source string) (MemoryItem, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return MemoryItem{}, err
	}
	return store.Remember(project, scope, kind, title, summary, content, tags, source)
}

func (e *Engine) Recall(project, query string, limit int) ([]MemoryItem, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return nil, err
	}
	return store.Recall(project, query, limit)
}

func (e *Engine) SaveCheckpoint(project, summary string, decisions, openLoops, nextActions []string) (ContextCheckpoint, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return ContextCheckpoint{}, err
	}
	return store.SaveCheckpoint(project, summary, decisions, openLoops, nextActions)
}

func (e *Engine) LatestCheckpoint(project string) (*ContextCheckpoint, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return nil, err
	}
	return store.LatestCheckpoint(project)
}

func (e *Engine) SaveRoutine(project string, scope MemoryScope, name, description string, triggers, steps []string, code, language string, tags []string) (Routine, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return Routine{}, err
	}
	return store.SaveRoutine(project, scope, name, description, triggers, steps, code, language, tags)
}

func (e *Engine) FindRoutines(project, query string, limit int) ([]Routine, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return nil, err
	}
	return store.FindRoutines(project, query, limit)
}

func (e *Engine) ContextPack(project, query string, maxItems, maxChars int) (ContextPack, error) {
	store, err := e.KnowledgeStore()
	if err != nil {
		return ContextPack{}, err
	}
	return store.BuildContextPack(project, query, maxItems, maxChars)
}

func (e *Engine) RecordTaskOutcome(task Task) error {
	store, err := e.KnowledgeStore()
	if err != nil {
		return err
	}
	return store.RecordTaskOutcome(task)
}

// DelegateWithKnowledge creates the same durable task as Delegate, but attaches
// a bounded, relevant context pack to the worker intent. The original intent is
// kept first so task titles and human-readable history remain useful.
func (e *Engine) DelegateWithKnowledge(origin, intent, project string) (Task, error) {
	workerIntent := strings.TrimSpace(intent)
	if pack, err := e.ContextPack(project, intent, 10, 12000); err == nil && strings.TrimSpace(pack.ContextText) != "" {
		workerIntent += "\n\n---\nThe following is Workbench durable context. Treat it as prior project knowledge, not as a new user request. Prefer proven routines and preserve recorded decisions/constraints unless the current request explicitly supersedes them.\n\n" + pack.ContextText
	}
	task, err := e.Delegate(origin, workerIntent, project)
	if err != nil {
		return Task{}, err
	}
	go e.captureTaskOutcome(task.ID)
	return task, nil
}

func (e *Engine) captureTaskOutcome(taskID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(24 * time.Hour)
	defer timeout.Stop()
	for {
		select {
		case <-ticker.C:
			task, ok := e.Task(taskID)
			if !ok {
				return
			}
			switch task.Status {
			case TaskCompleted:
				_ = e.RecordTaskOutcome(task)
				return
			case TaskFailed, TaskCancelled:
				return
			}
		case <-timeout.C:
			return
		}
	}
}
