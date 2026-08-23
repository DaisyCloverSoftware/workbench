package core

import (
	"context"
	"strings"
	"time"
)

type taskTelemetryReporter func(WorkProgress)

type taskTelemetryContextKey struct{}

const (
	maxTelemetryPhaseRunes = 160
	maxTelemetryUnitRunes  = 32
	maxTelemetryStages     = 1000
)

func withTaskTelemetryReporter(ctx context.Context, reporter taskTelemetryReporter) context.Context {
	if ctx == nil || reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, taskTelemetryContextKey{}, reporter)
}

func reportTaskTelemetry(ctx context.Context, progress WorkProgress) {
	if ctx == nil {
		return
	}
	reporter, _ := ctx.Value(taskTelemetryContextKey{}).(taskTelemetryReporter)
	if reporter == nil {
		return
	}
	if normalized, ok := normalizeTaskTelemetry(progress); ok {
		reporter(normalized)
	}
}

func normalizeTaskTelemetry(progress WorkProgress) (WorkProgress, bool) {
	progress.Phase = boundedTelemetryText(progress.Phase, maxTelemetryPhaseRunes)
	progress.Unit = boundedTelemetryText(progress.Unit, maxTelemetryUnitRunes)
	if progress.Phase == "" || LooksSecret(progress.Phase) || LooksSecret(progress.Unit) {
		return WorkProgress{}, false
	}
	switch progress.Kind {
	case ProgressMeasured:
		if progress.Total <= 0 || progress.Current < 0 || progress.Current > progress.Total {
			return WorkProgress{}, false
		}
		progress.Stage = 0
		progress.StageTotal = 0
		return progress, true
	case ProgressStages:
		if progress.StageTotal <= 0 || progress.StageTotal > maxTelemetryStages || progress.Stage <= 0 || progress.Stage > progress.StageTotal {
			return WorkProgress{}, false
		}
		progress.Current = 0
		progress.Total = 0
		progress.Unit = ""
		return progress, true
	default:
		return WorkProgress{}, false
	}
}

func boundedTelemetryText(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if maxRunes > 0 && len(runes) > maxRunes {
		value = string(runes[:maxRunes])
	}
	return value
}

func (e *Engine) updateTaskTelemetry(id string, progress WorkProgress) {
	progress, ok := normalizeTaskTelemetry(progress)
	if !ok {
		return
	}
	e.mu.Lock()
	i := e.taskIndexLocked(id)
	if i < 0 || (e.state.Tasks[i].Status != TaskRunning && e.state.Tasks[i].Status != TaskRouting) {
		e.mu.Unlock()
		return
	}
	e.state.Tasks[i].Progress = progress
	e.state.Tasks[i].UpdatedAt = time.Now()
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
}
