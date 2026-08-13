package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const workerMemoryPrefix = "WORKBENCH_MEMORY:"

// workerMemoryCandidate is intentionally project-only. Workers may suggest
// durable knowledge discovered while completing a task, but they never get to
// silently promote it to global scope.
type workerMemoryCandidate struct {
	Kind    KnowledgeKind `json:"kind"`
	Title   string        `json:"title"`
	Content string        `json:"content"`
	Tags    []string      `json:"tags,omitempty"`
}

// persistWorkerMemories removes structured memory-candidate lines from the
// human-facing report and saves valid, non-secret candidates at project scope.
// Memory persistence is best-effort: a bad candidate must never turn a
// successful coding task into a failed task.
func persistWorkerMemories(task Task, output string) string {
	clean, candidates := parseWorkerMemoryCandidates(output)
	for _, c := range candidates {
		_, _ = SaveKnowledge(KnowledgeItem{
			Scope:   ScopeProject,
			Project: task.ProjectPath,
			Kind:    c.Kind,
			Title:   c.Title,
			Content: c.Content,
			Tags:    c.Tags,
			Source:  task.ID,
		})
	}
	return clean
}

func parseWorkerMemoryCandidates(output string) (string, []workerMemoryCandidate) {
	lines := strings.Split(output, "\n")
	kept := make([]string, 0, len(lines))
	candidates := make([]workerMemoryCandidate, 0, 3)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, workerMemoryPrefix) {
			kept = append(kept, line)
			continue
		}

		// Candidate lines are control metadata, not completion-report prose. Strip
		// them even when malformed so provider mistakes do not leak noisy protocol
		// syntax into the user-facing result.
		if len(candidates) >= 3 {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(trimmed, workerMemoryPrefix))
		if raw == "" || len(raw) > 12<<10 {
			continue
		}
		var c workerMemoryCandidate
		dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			continue
		}
		var trailing any
		if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
			continue
		}
		c.Title = strings.TrimSpace(c.Title)
		c.Content = strings.TrimSpace(c.Content)
		if c.Title == "" || c.Content == "" || len(c.Title) > 240 || len(c.Content) > 8000 {
			continue
		}
		if !validWorkerMemoryKind(c.Kind) {
			continue
		}
		if LooksSecret(c.Title + "\n" + c.Content + "\n" + strings.Join(c.Tags, "\n")) {
			continue
		}
		if len(c.Tags) > 12 {
			c.Tags = c.Tags[:12]
		}
		c.Tags = normalizeTags(c.Tags)
		candidates = append(candidates, c)
	}

	return strings.TrimSpace(strings.Join(kept, "\n")), candidates
}

func validWorkerMemoryKind(kind KnowledgeKind) bool {
	switch kind {
	case KindFact, KindDecision, KindConstraint, KindPattern, KindRoutine, KindCode:
		return true
	default:
		return false
	}
}
