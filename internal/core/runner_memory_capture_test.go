package core

import (
	"strings"
	"testing"
)

func TestPersistWorkerMemoriesSavesProjectKnowledgeAndCleansReport(t *testing.T) {
	isolateKnowledgeConfig(t)
	task := Task{ID: "task-123", ProjectPath: "/workspace/sample"}
	out := "done\nWORKBENCH_MEMORY: {\"kind\":\"routine\",\"title\":\"Retry verification\",\"content\":\"Run the focused retry test before the full suite.\",\"tags\":[\"retry\",\"test\"]}\nverified"

	clean := persistWorkerMemories(task, out)
	if strings.Contains(clean, workerMemoryPrefix) || clean != "done\nverified" {
		t.Fatalf("unexpected clean report: %q", clean)
	}
	items, err := SearchKnowledge(task.ProjectPath, "retry verification", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one saved memory, got %#v", items)
	}
	if items[0].Scope != ScopeProject || items[0].Project != task.ProjectPath || items[0].Kind != KindRoutine || items[0].Source != task.ID {
		t.Fatalf("unexpected saved memory: %#v", items[0])
	}
}

func TestParseWorkerMemoryCandidatesRejectsUnsafeMalformedAndGlobalControl(t *testing.T) {
	secretish := "token: " + strings.Repeat("x", 12)
	out := strings.Join([]string{
		"report",
		`WORKBENCH_MEMORY: {"kind":"routine","title":"Good","content":"Reuse the tested parser helper."}`,
		`WORKBENCH_MEMORY: {"kind":"routine","title":"Bad secret","content":"` + secretish + `"}`,
		`WORKBENCH_MEMORY: {"kind":"routine","title":"Unknown field","content":"x","scope":"global"}`,
		`WORKBENCH_MEMORY: not-json`,
		"done",
	}, "\n")

	clean, candidates := parseWorkerMemoryCandidates(out)
	if len(candidates) != 1 || candidates[0].Title != "Good" {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
	if strings.Contains(clean, workerMemoryPrefix) || clean != "report\ndone" {
		t.Fatalf("unexpected clean output: %q", clean)
	}
}

func TestParseWorkerMemoryCandidatesCapsAtThree(t *testing.T) {
	var lines []string
	for _, title := range []string{"one", "two", "three", "four"} {
		lines = append(lines, `WORKBENCH_MEMORY: {"kind":"fact","title":"`+title+`","content":"durable content"}`)
	}
	_, candidates := parseWorkerMemoryCandidates(strings.Join(lines, "\n"))
	if len(candidates) != 3 {
		t.Fatalf("expected three candidates, got %d", len(candidates))
	}
}
