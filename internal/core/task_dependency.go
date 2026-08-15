package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const githubActionsWaitPrefix = "WORKBENCH_WAIT_GITHUB_ACTIONS:"

const (
	initialDependencyCheckDelay = 20 * time.Second
	maxDependencyAttemptNotes   = 200
)

type githubActionsWaitEnvelope struct {
	Repository string `json:"repository"`
	RunID      int64  `json:"run_id"`
}

type dependencyObservation struct {
	Status       string
	Conclusion   string
	WorkflowName string
	Completed    bool
}

// tryDelegateDeferredDependency recognises the Workbench skill's compact wait
// envelope. This keeps the public MCP surface backwards-compatible while giving
// connected AIs a durable way to say "resume this after CI" instead of holding
// a worker open or relying on chat memory to remember a polling loop.
func (e *Engine) tryDelegateDeferredDependency(origin, intent, project string) (Task, bool, error) {
	dependency, continuation, matched, err := parseDeferredGitHubActionsIntent(intent, time.Now().UTC())
	if !matched {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, true, err
	}
	project, err = canonicalProjectSelection(project)
	if err != nil {
		return Task{}, true, errors.New("choose a valid project folder first")
	}

	now := time.Now().UTC()
	t := Task{
		ID:           newID("task"),
		CreatedAt:    now,
		UpdatedAt:    now,
		Origin:       origin,
		Title:        TaskTitle(continuation),
		Intent:       continuation,
		ProjectPath:  project,
		Status:       TaskWaitingDependency,
		RouteReason:  "waiting on an external dependency in the background; no coding worker is reserved",
		Dependency:   dependency,
		ConsumesWork: false,
		Attempts: []string{
			fmt.Sprintf("Workbench: monitoring GitHub Actions run %d with progressive backoff; first check after %s", dependency.RunID, dependency.NextCheckAt.Format(time.RFC3339)),
		},
	}

	e.mu.Lock()
	for _, existing := range e.state.Tasks {
		if existing.Status != TaskWaitingDependency || existing.Dependency == nil {
			continue
		}
		if sameProjectPath(existing.ProjectPath, project) &&
			existing.Dependency.Kind == DependencyGitHubActions &&
			existing.Dependency.RunID == dependency.RunID &&
			strings.EqualFold(existing.Dependency.Repository, dependency.Repository) &&
			strings.TrimSpace(existing.Intent) == strings.TrimSpace(continuation) {
			existing.Attempts = append([]string(nil), existing.Attempts...)
			existing.Review = cloneTaskReviewResult(existing.Review)
			existing.Dependency = cloneTaskDependency(existing.Dependency)
			e.mu.Unlock()
			return existing, true, nil
		}
	}
	touchProjectState(&e.state, project)
	e.state.Tasks = append([]Task{t}, e.state.Tasks...)
	st := cloneState(e.state)
	e.mu.Unlock()

	if err := e.store.Save(st); err != nil {
		return Task{}, true, err
	}
	e.notify()
	go e.scheduleTaskDependencyCheck(t.ID, dependency.NextCheckAt)
	return t, true, nil
}

func parseDeferredGitHubActionsIntent(intent string, now time.Time) (*TaskDependency, string, bool, error) {
	intent = strings.TrimSpace(intent)
	if intent == "" {
		return nil, "", false, nil
	}
	// The Git relay adds a bounded correlation marker before delegated intents.
	// Strip only that well-formed marker so the documented wait envelope remains
	// usable through both direct MCP and relay transports without weakening the
	// envelope parser for arbitrary prefixes.
	if strings.HasPrefix(intent, "[relay:") {
		if end := strings.Index(intent, "] "); end > len("[relay:") && end <= 96 {
			relayID := intent[len("[relay:"):end]
			valid := relayID != ""
			for _, r := range relayID {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
					continue
				}
				valid = false
				break
			}
			if valid {
				intent = strings.TrimSpace(intent[end+2:])
			}
		}
	}
	lines := strings.SplitN(strings.ReplaceAll(intent, "\r\n", "\n"), "\n", 2)
	header := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(header, githubActionsWaitPrefix) {
		return nil, "", false, nil
	}
	if len(lines) != 2 || strings.TrimSpace(lines[1]) == "" {
		return nil, "", true, errors.New("GitHub Actions dependency wait requires a continuation intent after the wait envelope")
	}

	raw := strings.TrimSpace(strings.TrimPrefix(header, githubActionsWaitPrefix))
	dec := json.NewDecoder(bytes.NewBufferString(raw))
	dec.DisallowUnknownFields()
	var envelope githubActionsWaitEnvelope
	if err := dec.Decode(&envelope); err != nil {
		return nil, "", true, fmt.Errorf("invalid GitHub Actions wait envelope: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, "", true, errors.New("invalid GitHub Actions wait envelope: trailing data")
	}
	envelope.Repository = strings.TrimSpace(envelope.Repository)
	if !validGitHubRepoSlug(envelope.Repository) {
		return nil, "", true, errors.New("GitHub Actions dependency repository must be an owner/repository slug")
	}
	if envelope.RunID <= 0 {
		return nil, "", true, errors.New("GitHub Actions dependency run_id must be positive")
	}

	now = now.UTC()
	dependency := &TaskDependency{
		Kind:        DependencyGitHubActions,
		Reason:      "GitHub Actions run must finish before continuation",
		Repository:  envelope.Repository,
		RunID:       envelope.RunID,
		State:       "pending",
		StartedAt:   now,
		NextCheckAt: now.Add(initialDependencyCheckDelay),
	}
	return dependency, strings.TrimSpace(lines[1]), true, nil
}

func cloneTaskDependency(dependency *TaskDependency) *TaskDependency {
	if dependency == nil {
		return nil
	}
	cloned := *dependency
	return &cloned
}

func dependencyPollDelay(checkCount int, state string) time.Duration {
	if checkCount < 1 {
		checkCount = 1
	}
	state = strings.ToLower(strings.TrimSpace(state))
	var schedule []time.Duration
	switch state {
	case "in_progress":
		schedule = []time.Duration{20 * time.Second, 30 * time.Second, 45 * time.Second, time.Minute, 90 * time.Second, 2 * time.Minute}
	case "probe_unavailable":
		schedule = []time.Duration{time.Minute, 2 * time.Minute, 3 * time.Minute, 5 * time.Minute, 10 * time.Minute}
	default:
		// Queued/waiting CI tends to be the bottleneck. Back off more quickly so a
		// busy shared runner is observed without being hammered.
		schedule = []time.Duration{30 * time.Second, 45 * time.Second, time.Minute, 90 * time.Second, 2 * time.Minute, 2 * time.Minute}
	}
	index := checkCount - 1
	if index >= len(schedule) {
		index = len(schedule) - 1
	}
	return schedule[index]
}

func (e *Engine) scheduleTaskDependencyCheck(taskID string, checkAt time.Time) {
	delay := time.Until(checkAt)
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
	}
	e.fireTaskDependencyCheck(taskID, checkAt)
}

func (e *Engine) fireTaskDependencyCheck(taskID string, expected time.Time) {
	e.mu.RLock()
	i := e.taskIndexLocked(taskID)
	if i < 0 || e.state.Tasks[i].Status != TaskWaitingDependency || e.state.Tasks[i].Dependency == nil || !e.state.Tasks[i].Dependency.NextCheckAt.Equal(expected) {
		e.mu.RUnlock()
		return
	}
	dependency := cloneTaskDependency(e.state.Tasks[i].Dependency)
	e.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	observation, probeErr := probeTaskDependency(ctx, dependency)
	cancel()

	now := time.Now().UTC()
	e.mu.Lock()
	i = e.taskIndexLocked(taskID)
	if i < 0 || e.state.Tasks[i].Status != TaskWaitingDependency || e.state.Tasks[i].Dependency == nil || !e.state.Tasks[i].Dependency.NextCheckAt.Equal(expected) {
		e.mu.Unlock()
		return
	}
	task := &e.state.Tasks[i]
	dep := task.Dependency
	previousState := strings.TrimSpace(dep.State)
	dep.CheckCount++
	dep.LastCheckedAt = now

	if probeErr != nil {
		dep.State = "probe_unavailable"
		dep.Conclusion = ""
		dep.NextCheckAt = now.Add(dependencyPollDelay(dep.CheckCount, dep.State))
		task.RouteReason = "dependency probe is temporarily unavailable; Workbench is backing off and will keep monitoring without reserving a coding worker"
		task.UpdatedAt = now
		if shouldRecordDependencyCheck(dep.CheckCount, previousState, dep.State, 5) {
			appendBoundedTaskAttempt(task, fmt.Sprintf("Workbench: GitHub Actions dependency probe unavailable; next background check %s", dep.NextCheckAt.Format(time.RFC3339)))
		}
		st := cloneState(e.state)
		next := dep.NextCheckAt
		e.mu.Unlock()
		if err := e.store.Save(st); err != nil {
			e.finishFailed(taskID, "Workbench could not persist the external dependency watch: "+err.Error())
			return
		}
		e.notify()
		go e.scheduleTaskDependencyCheck(taskID, next)
		return
	}

	dep.State = strings.ToLower(strings.TrimSpace(observation.Status))
	if dep.State == "" {
		dep.State = "pending"
	}
	dep.Conclusion = strings.ToLower(strings.TrimSpace(observation.Conclusion))
	if !observation.Completed {
		dep.NextCheckAt = now.Add(dependencyPollDelay(dep.CheckCount, dep.State))
		task.RouteReason = "waiting on an external dependency in the background; other Workbench tasks can continue"
		task.UpdatedAt = now
		if shouldRecordDependencyCheck(dep.CheckCount, previousState, dep.State, 10) {
			appendBoundedTaskAttempt(task, fmt.Sprintf("Workbench: GitHub Actions run %d is %s; next background check %s", dep.RunID, dependencyStateLabel(dep.State), dep.NextCheckAt.Format(time.RFC3339)))
		}
		st := cloneState(e.state)
		next := dep.NextCheckAt
		e.mu.Unlock()
		if err := e.store.Save(st); err != nil {
			e.finishFailed(taskID, "Workbench could not persist the external dependency watch: "+err.Error())
			return
		}
		e.notify()
		go e.scheduleTaskDependencyCheck(taskID, next)
		return
	}

	summary := dependencyCompletionSummary(dep, observation)
	task.DependencyResult = summary
	task.Dependency = nil
	task.Status = TaskQueued
	task.ProviderID = ""
	task.RouteReason = ""
	task.ConsumesWork = false
	task.RetryAt = nil
	task.FinishedAt = nil
	task.UpdatedAt = now
	task.Intent = strings.TrimSpace(task.Intent) + "\n\nWorkbench dependency update:\n" + summary + "\nContinue the original task now. If CI failed, inspect the failure and fix it autonomously when the repair is within the existing scope. If it succeeded, continue the remaining in-scope work. Only require human attention for a genuine decision or authority boundary."
	appendBoundedTaskAttempt(task, "Workbench: external dependency became terminal; resuming the task automatically without waiting for a chat session")
	st := cloneState(e.state)
	e.mu.Unlock()

	if err := e.store.Save(st); err != nil {
		e.finishFailed(taskID, "Workbench could not persist the completed dependency result: "+err.Error())
		return
	}
	e.notify()
	go e.execute(taskID)
}

func shouldRecordDependencyCheck(checkCount int, previousState, currentState string, every int) bool {
	if checkCount <= 1 || previousState != currentState {
		return true
	}
	return every > 0 && checkCount%every == 0
}

func appendBoundedTaskAttempt(task *Task, text string) {
	if task == nil || strings.TrimSpace(text) == "" {
		return
	}
	task.Attempts = append(task.Attempts, strings.TrimSpace(text))
	if len(task.Attempts) > maxDependencyAttemptNotes {
		task.Attempts = append([]string(nil), task.Attempts[len(task.Attempts)-maxDependencyAttemptNotes:]...)
	}
}

func probeTaskDependency(ctx context.Context, dependency *TaskDependency) (dependencyObservation, error) {
	if dependency == nil {
		return dependencyObservation{}, errors.New("dependency is missing")
	}
	switch dependency.Kind {
	case DependencyGitHubActions:
		return probeGitHubActionsRunWithRunner(ctx, dependency.Repository, dependency.RunID, runGHCommand)
	default:
		return dependencyObservation{}, fmt.Errorf("unsupported dependency kind %q", dependency.Kind)
	}
}

func probeGitHubActionsRunWithRunner(ctx context.Context, repository string, runID int64, run ghCommandRunner) (dependencyObservation, error) {
	if !validGitHubRepoSlug(repository) || runID <= 0 {
		return dependencyObservation{}, errors.New("invalid GitHub Actions dependency locator")
	}
	out, err := run(ctx,
		"run", "view", strconv.FormatInt(runID, 10),
		"--repo", repository,
		"--json", "databaseId,status,conclusion,workflowName",
	)
	if err != nil {
		return dependencyObservation{}, errors.New("GitHub Actions dependency is temporarily unavailable")
	}
	var response struct {
		DatabaseID   int64  `json:"databaseId"`
		Status       string `json:"status"`
		Conclusion   string `json:"conclusion"`
		WorkflowName string `json:"workflowName"`
	}
	if err := json.Unmarshal(out, &response); err != nil || response.DatabaseID != runID {
		return dependencyObservation{}, errors.New("GitHub Actions dependency returned an invalid response")
	}
	status := strings.ToLower(strings.TrimSpace(response.Status))
	return dependencyObservation{
		Status:       status,
		Conclusion:   strings.ToLower(strings.TrimSpace(response.Conclusion)),
		WorkflowName: strings.TrimSpace(response.WorkflowName),
		Completed:    status == "completed",
	}, nil
}

func dependencyStateLabel(state string) string {
	state = strings.TrimSpace(strings.ReplaceAll(state, "_", " "))
	if state == "" {
		return "pending"
	}
	return state
}

func dependencyCompletionSummary(dependency *TaskDependency, observation dependencyObservation) string {
	name := strings.TrimSpace(observation.WorkflowName)
	if name == "" {
		name = "workflow"
	}
	conclusion := strings.TrimSpace(observation.Conclusion)
	if conclusion == "" {
		conclusion = "completed"
	}
	return fmt.Sprintf("GitHub Actions run %d (%s) completed with conclusion %s.", dependency.RunID, name, conclusion)
}
