package core

import (
	"errors"
	"strings"
	"time"
)

// ValidateWorkProgress sanitises one structured progress update before it is
// persisted or shown. Progress strings are deliberately tiny and secret-checked
// because they may cross the same dashboard/relay surfaces as other job state.
func ValidateWorkProgress(progress WorkProgress) (WorkProgress, error) {
	progress.Kind = WorkProgressKind(strings.ToLower(strings.TrimSpace(string(progress.Kind))))
	progress.Unit = strings.TrimSpace(progress.Unit)
	progress.StageName = strings.TrimSpace(progress.StageName)
	if strings.ContainsAny(progress.Unit+progress.StageName, "\x00\r\n") {
		return WorkProgress{}, errors.New("progress labels must be single-line text")
	}
	if len(progress.Unit) > 32 || len(progress.StageName) > 160 {
		return WorkProgress{}, errors.New("progress label is too long")
	}
	if LooksSecret(progress.Unit) || LooksSecret(progress.StageName) {
		return WorkProgress{}, errors.New("progress label resembles secret material")
	}

	switch progress.Kind {
	case "", WorkProgressNone:
		return WorkProgress{Kind: WorkProgressNone}, nil
	case WorkProgressMeasured:
		if progress.Total <= 0 || progress.Current < 0 || progress.Current > progress.Total {
			return WorkProgress{}, errors.New("measured progress requires 0 <= current <= total and total > 0")
		}
		return WorkProgress{
			Kind:    WorkProgressMeasured,
			Current: progress.Current,
			Total:   progress.Total,
			Unit:    progress.Unit,
		}, nil
	case WorkProgressStages:
		if progress.StageTotal <= 0 || progress.Stage < 0 || progress.Stage > progress.StageTotal {
			return WorkProgress{}, errors.New("stage progress requires 0 <= stage <= stage_total and stage_total > 0")
		}
		return WorkProgress{
			Kind:       WorkProgressStages,
			Stage:      progress.Stage,
			StageTotal: progress.StageTotal,
			StageName:  progress.StageName,
		}, nil
	case WorkProgressIndeterminate:
		return WorkProgress{Kind: WorkProgressIndeterminate, StageName: progress.StageName}, nil
	default:
		return WorkProgress{}, errors.New("unsupported progress kind")
	}
}

// UpdateTaskProgress records truthful structured progress for an unfinished
// task. It does not alter scheduling state or claim that an indeterminate job
// has a percentage.
func (e *Engine) UpdateTaskProgress(taskID string, progress WorkProgress) error {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return errors.New("task id is empty")
	}
	progress, err := ValidateWorkProgress(progress)
	if err != nil {
		return err
	}

	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	if !taskBlocksProjectRemoval(e.state.Tasks[i].Status) {
		e.mu.Unlock()
		return errors.New("terminal task progress cannot be changed")
	}
	e.state.Tasks[i].Progress = progress
	e.state.Tasks[i].UpdatedAt = time.Now().UTC()
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}
