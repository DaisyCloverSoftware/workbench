package core

import (
	"fmt"
	"strings"
)

const workerMemoryBudget = 16000

// BuildWorkerPromptFromStoredKnowledge assembles an autonomous worker prompt from
// the current task plus the latest compact project context and the most relevant
// project/global knowledge. Memory lookup failures never block execution; the
// repository remains the source of truth.
func BuildWorkerPromptFromStoredKnowledge(task Task) string {
	memories, _ := SearchKnowledge(task.ProjectPath, task.Intent, 8)
	var capsule *ContextCapsule
	if c, ok, err := LatestContextCapsule(task.ProjectPath); err == nil && ok {
		capsule = &c
	}
	return BuildWorkerPromptWithKnowledge(task, memories, capsule)
}

// BuildWorkerPromptWithKnowledge adds a bounded, explicitly advisory memory layer
// to the ordinary worker prompt. Repository state remains authoritative: saved
// routines, decisions and code references are there to prevent needless reinvention,
// not to override what the worker can verify in the current tree.
func BuildWorkerPromptWithKnowledge(task Task, memories []KnowledgeItem, capsule *ContextCapsule) string {
	base := BuildWorkerPrompt(task)
	var extra strings.Builder

	if capsule != nil && strings.TrimSpace(capsule.ID) != "" {
		extra.WriteString("\nCurrent compact context (resume from this instead of reconstructing old conversation):\n")
		extra.WriteString("- Objective: " + compactMemoryText(capsule.Objective, 1800) + "\n")
		extra.WriteString("- State: " + compactMemoryText(capsule.State, 3500) + "\n")
		if len(capsule.Decisions) > 0 {
			extra.WriteString("- Decisions: " + compactMemoryText(strings.Join(capsule.Decisions, "; "), 2500) + "\n")
		}
		if len(capsule.Constraints) > 0 {
			extra.WriteString("- Constraints: " + compactMemoryText(strings.Join(capsule.Constraints, "; "), 2500) + "\n")
		}
		if strings.TrimSpace(capsule.NextAction) != "" {
			extra.WriteString("- Next action: " + compactMemoryText(capsule.NextAction, 1800) + "\n")
		}
	}

	if len(memories) > 0 {
		extra.WriteString("\nRelevant Workbench memory (advisory; verify it against the current repository):\n")
		for i, item := range memories {
			if i >= 8 || extra.Len() >= workerMemoryBudget {
				break
			}
			remaining := workerMemoryBudget - extra.Len()
			if remaining <= 256 {
				break
			}
			contentLimit := 3200
			if contentLimit > remaining-160 {
				contentLimit = remaining - 160
			}
			if contentLimit < 128 {
				break
			}
			extra.WriteString(fmt.Sprintf("- [%s/%s] %s: %s\n", item.Scope, item.Kind, compactMemoryText(item.Title, 300), compactMemoryText(item.Content, contentLimit)))
		}
		extra.WriteString("Reuse before rebuilding: prefer a verified saved routine, pattern, or existing code reference over creating another equivalent implementation.\n")
	}

	return base + extra.String()
}

func compactMemoryText(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 || len(s) <= limit {
		return s
	}
	if limit < len(" [truncated]") {
		return s[:limit]
	}
	return strings.TrimSpace(s[:limit-len(" [truncated]")]) + " [truncated]"
}
