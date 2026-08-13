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
	memories, _ := SearchKnowledge(task.ProjectPath, task.Intent, 16)
	memories = FilterActiveReusableKnowledge(memories)
	if len(memories) > 8 {
		memories = memories[:8]
	}
	for _, item := range memories {
		_ = MarkKnowledgeUsed(item.ID)
	}
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
	base += "\nReusable-asset memory rule: for WORKBENCH_MEMORY items of kind routine or code, include an optional verification field only when you actually ran the stated build/test/check evidence. Changed routine/code content becomes a new Workbench asset version; do not emit a cosmetic rewrite as a new version.\n"
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
			label := fmt.Sprintf("%s/%s", item.Scope, item.Kind)
			if item.Kind == KindRoutine || item.Kind == KindCode {
				label += fmt.Sprintf(" v%d", ReusableAssetVersion(item))
				if ReusableAssetVerified(item) {
					label += " verified"
				}
			}
			extra.WriteString(fmt.Sprintf("- [%s] %s: %s\n", label, compactMemoryText(item.Title, 300), compactMemoryText(item.Content, contentLimit)))
		}
		extra.WriteString("Reuse before rebuilding: prefer the newest verified saved routine/code asset, then a relevant pattern or code reference, over creating another equivalent implementation. Repository state remains authoritative.\n")
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
