package core

// ResumeInterruptedTasks restarts durable queued/routing/running work after a
// Workbench process restart. Tasks already waiting for human attention, plus
// terminal tasks, remain where they were.
func (e *Engine) ResumeInterruptedTasks() error {
	e.mu.Lock()
	ids := recoverInterruptedTasks(&e.state)
	st := cloneState(e.state)
	e.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	for _, id := range ids {
		go e.execute(id)
	}
	return nil
}
