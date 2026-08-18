package core

import "errors"

// TaskCanArchive is deliberately limited to terminal records. Active work,
// dependency watches and human-attention tasks remain visible/actionable and
// cannot be hidden by the history UI.
func TaskCanArchive(status TaskStatus) bool {
	switch status {
	case TaskCompleted, TaskFailed, TaskCancelled:
		return true
	default:
		return false
	}
}

// SetTaskArchived changes only the history presentation flag. The durable task
// record, output, attempts, review metadata and execution timestamps remain in
// State so archive is fully reversible and never becomes a deletion mechanism.
// In particular UpdatedAt is not rewritten: filing old work must not make it
// look like the task itself just ran again when history is restored.
func (e *Engine) SetTaskArchived(taskID string, archived bool) error {
	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	if archived && !TaskCanArchive(e.state.Tasks[i].Status) {
		e.mu.Unlock()
		return errors.New("only completed, failed or cancelled tasks can be archived")
	}
	if e.state.Tasks[i].Archived == archived {
		e.mu.Unlock()
		return nil
	}
	e.state.Tasks[i].Archived = archived
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}
