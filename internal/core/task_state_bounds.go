package core

import (
	"errors"
	"fmt"
	"strings"
)

const (
	maxDurableTasks       = 1000
	maxTaskIntentBytes    = 128 << 10
	maxTaskAnswerBytes    = 64 << 10
	maxTaskProjectBytes   = 32 << 10
	maxTaskOutputBytes    = 512 << 10
	maxTaskErrorBytes     = 128 << 10
	maxTaskAttentionBytes = 16 << 10
	maxTaskRouteBytes     = 8 << 10
	maxTaskAttempts       = 100
	maxTaskAttemptBytes   = 4 << 10
)

const durableTaskTruncation = "\n… [truncated by Workbench]"

// normalizeDurableTaskState keeps operational reports from making the local
// state file grow without bound while preserving semantic inputs exactly. Task
// intent, project identity and human answers are rejected when implausibly large
// rather than silently truncated because changing those fields could change what
// a resumed autonomous task does.
func normalizeDurableTaskState(st *State) error {
	if st == nil {
		return nil
	}
	if len(st.Tasks) > maxDurableTasks {
		st.Tasks = append([]Task(nil), st.Tasks[:maxDurableTasks]...)
	}
	for i := range st.Tasks {
		t := &st.Tasks[i]
		if len(t.Intent) > maxTaskIntentBytes {
			return fmt.Errorf("task %s intent exceeds durable state limit", t.ID)
		}
		if len(t.HumanAnswer) > maxTaskAnswerBytes {
			return fmt.Errorf("task %s human answer exceeds durable state limit", t.ID)
		}
		if len(t.ProjectPath) > maxTaskProjectBytes {
			return fmt.Errorf("task %s project path exceeds durable state limit", t.ID)
		}
		t.Output = truncateDurableTaskText(t.Output, maxTaskOutputBytes)
		t.Error = truncateDurableTaskText(t.Error, maxTaskErrorBytes)
		t.AttentionQuestion = truncateDurableTaskText(t.AttentionQuestion, maxTaskAttentionBytes)
		t.RouteReason = truncateDurableTaskText(t.RouteReason, maxTaskRouteBytes)
		if len(t.Attempts) > maxTaskAttempts {
			// Keep the most recent routing/recovery evidence; old attempts are
			// diagnostic history rather than task semantics.
			t.Attempts = append([]string(nil), t.Attempts[len(t.Attempts)-maxTaskAttempts:]...)
		}
		for j := range t.Attempts {
			t.Attempts[j] = truncateDurableTaskText(t.Attempts[j], maxTaskAttemptBytes)
		}
	}
	return nil
}

func truncateDurableTaskText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	marker := durableTaskTruncation
	if limit <= len(marker) {
		return value[:limit]
	}
	return strings.TrimSpace(value[:limit-len(marker)]) + marker
}

func validateDurableTaskState(st State) error {
	copy := st
	if err := normalizeDurableTaskState(&copy); err != nil {
		return err
	}
	if len(copy.Tasks) > maxDurableTasks {
		return errors.New("durable task count exceeds limit")
	}
	return nil
}
