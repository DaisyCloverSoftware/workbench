package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

var BuildDefaultOpenClawSSHHost string

type Engine struct {
	mu        sync.RWMutex
	store     *Store
	state     State
	providers []Provider
	cancel    map[string]context.CancelFunc
	onChange  []func()
}

func NewEngine(store *Store) (*Engine, error) {
	st, err := store.Load()
	if err != nil {
		return nil, err
	}
	if st.Preferences.MCPPort == 0 {
		st.Preferences.MCPPort = 8765
	}
	if strings.TrimSpace(st.Preferences.MCPToken) == "" {
		st.Preferences.MCPToken = NewToken()
	}
	if strings.TrimSpace(st.Preferences.OpenClawSSHHost) == "" && strings.TrimSpace(BuildDefaultOpenClawSSHHost) != "" {
		st.Preferences.OpenClawSSHHost = strings.TrimSpace(BuildDefaultOpenClawSSHHost)
	}
	e := &Engine{store: store, state: st, cancel: map[string]context.CancelFunc{}}
	e.providers = ScanProviders()
	_ = e.store.Save(st)
	return e, nil
}

func (e *Engine) Subscribe(fn func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onChange = append(e.onChange, fn)
}

func (e *Engine) notify() {
	e.mu.RLock()
	fns := append([]func(){}, e.onChange...)
	e.mu.RUnlock()
	for _, fn := range fns {
		if fn != nil {
			fn()
		}
	}
}

func (e *Engine) State() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return cloneState(e.state)
}

func cloneState(s State) State {
	x := s
	x.Projects = append([]Project(nil), s.Projects...)
	x.Tasks = append([]Task(nil), s.Tasks...)
	for i := range x.Tasks {
		x.Tasks[i].Attempts = append([]string(nil), s.Tasks[i].Attempts...)
		x.Tasks[i].Review = cloneTaskReviewResult(s.Tasks[i].Review)
	}
	x.Secrets = append([]SecretRef(nil), s.Secrets...)
	return x
}

func (e *Engine) Providers() []Provider {
	e.mu.RLock()
	providers := append([]Provider(nil), e.providers...)
	e.mu.RUnlock()
	return ApplyProviderHealth(providers, time.Now())
}

func (e *Engine) RescanProviders() {
	_ = ClearAllProviderCooldowns()
	p := ScanProviders()
	e.mu.Lock()
	e.providers = p
	e.mu.Unlock()
	e.notify()
}

func (e *Engine) SaveNotes(project, notes string) error {
	e.mu.Lock()
	if err := saveProjectNotesState(&e.state, project, notes); err != nil {
		e.mu.Unlock()
		return err
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

func (e *Engine) SavePreferences(p Preferences) error {
	if p.MCPPort == 0 {
		p.MCPPort = 8765
	}
	if p.MCPToken == "" {
		p.MCPToken = NewToken()
	}
	e.mu.Lock()
	e.state.Preferences = p
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

func (e *Engine) AddSecret(ref SecretRef) error {
	e.mu.Lock()
	for i := range e.state.Secrets {
		if strings.EqualFold(e.state.Secrets[i].Name, ref.Name) {
			e.state.Secrets[i] = ref
			st := cloneState(e.state)
			e.mu.Unlock()
			if err := e.store.Save(st); err != nil {
				return err
			}
			e.notify()
			return nil
		}
	}
	e.state.Secrets = append(e.state.Secrets, ref)
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	return nil
}

func (e *Engine) Delegate(origin, intent, project string) (Task, error) {
	if strings.TrimSpace(intent) == "" {
		return Task{}, errors.New("tell Workbench what outcome you want")
	}
	project, err := canonicalProjectSelection(project)
	if err != nil {
		return Task{}, errors.New("choose a valid project folder first")
	}
	now := time.Now()
	t := Task{ID: newID("task"), CreatedAt: now, UpdatedAt: now, Origin: origin, Title: TaskTitle(intent), Intent: intent, ProjectPath: project, Status: TaskQueued}
	e.mu.Lock()
	touchProjectState(&e.state, project)
	e.state.Tasks = append([]Task{t}, e.state.Tasks...)
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return Task{}, err
	}
	e.notify()
	go e.execute(t.ID)
	return t, nil
}

func (e *Engine) ResolveAttention(taskID, answer string) error {
	if strings.TrimSpace(answer) == "" {
		return errors.New("answer is empty")
	}
	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	if e.state.Tasks[i].Status != TaskNeedsAttention {
		e.mu.Unlock()
		return errors.New("task is not waiting for attention")
	}
	e.state.Tasks[i].HumanAnswer = answer
	e.state.Tasks[i].AttentionQuestion = ""
	e.state.Tasks[i].Status = TaskQueued
	e.state.Tasks[i].UpdatedAt = time.Now()
	st := cloneState(e.state)
	e.mu.Unlock()
	if err := e.store.Save(st); err != nil {
		return err
	}
	e.notify()
	go e.execute(taskID)
	return nil
}

func (e *Engine) Cancel(taskID string) error {
	e.mu.Lock()
	if c := e.cancel[taskID]; c != nil {
		c()
	}
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return errors.New("task not found")
	}
	now := time.Now()
	e.state.Tasks[i].Status = TaskCancelled
	e.state.Tasks[i].UpdatedAt = now
	e.state.Tasks[i].FinishedAt = &now
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
	return nil
}

func (e *Engine) Task(id string) (Task, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, t := range e.state.Tasks {
		if t.ID == id {
			t.Attempts = append([]string(nil), t.Attempts...)
			t.Review = cloneTaskReviewResult(t.Review)
			return t, true
		}
	}
	return Task{}, false
}

func (e *Engine) execute(taskID string) {
	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	if old := e.cancel[taskID]; old != nil {
		old()
	}
	e.cancel[taskID] = cancel
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		cancel()
		return
	}
	t := e.state.Tasks[i]
	e.state.Tasks[i].Status = TaskRouting
	e.state.Tasks[i].UpdatedAt = time.Now()
	providers := append([]Provider(nil), e.providers...)
	prefs := e.state.Preferences
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
	defer func() { e.mu.Lock(); delete(e.cancel, taskID); e.mu.Unlock(); cancel() }()

	candidates := routeCandidates(providers, prefs, t)
	candidates, cooling := FilterProviderCooldowns(candidates, time.Now())
	for _, skipped := range cooling {
		e.appendAttempt(taskID, skipped)
	}
	if len(candidates) == 0 {
		if len(cooling) > 0 {
			e.finishFailed(taskID, "All eligible coding workers are temporarily cooling down after recent provider-level failures. Use Rescan after fixing provider setup, or retry after the cooldown.\n\n"+strings.Join(cooling, "\n"))
			return
		}
		e.finishFailed(taskID, "No eligible coding worker is connected. Connect Gemini, Copilot, Claude, OpenClaw, or Codex; Workbench will keep metered APIs disabled unless you opt in.")
		return
	}

	errorsSeen := append([]string(nil), cooling...)
	for _, p := range candidates {
		if ctx.Err() != nil {
			return
		}
		e.updateRunning(taskID, p)
		current, ok := e.Task(taskID)
		if !ok {
			return
		}
		runCtx, runCancel := context.WithTimeout(ctx, 45*time.Minute)
		res, err := RunProviderIsolated(runCtx, p, current, prefs)
		runCancel()
		record, coolingNow := RecordProviderRunOutcome(p.ID, res, err)
		attempt := fmt.Sprintf("%s: %s", p.Name, attemptSummary(res, err))
		if coolingNow {
			attempt += fmt.Sprintf("; cooldown until %s (%s)", record.CooldownUntil.UTC().Format(time.RFC3339), record.Reason)
		}
		e.appendAttempt(taskID, attempt)
		if ctx.Err() != nil {
			return
		}
		if strings.TrimSpace(res.Attention) != "" {
			e.finishAttention(taskID, p, res)
			e.sendAttentionNotification(taskID, res.Attention, prefs.NotificationCommand)
			return
		}
		if err == nil {
			e.finishCompleted(taskID, p, res)
			return
		}
		errorsSeen = append(errorsSeen, attempt)
		// To preserve scarce Work, only escalate past included workers if allowed by policy.
		if p.Cost == CostScarce && prefs.AvoidWorkUsage {
			// We reached Work only because no lower-cost candidate succeeded; trying once is intentional.
		}
	}
	e.finishFailed(taskID, "Every eligible worker failed.\n\n"+strings.Join(errorsSeen, "\n"))
}

func routeCandidates(providers []Provider, prefs Preferences, t Task) []Provider {
	var out []Provider
	for _, p := range providers {
		if !p.Installed || !p.Authenticated || !p.CanWrite || strings.TrimSpace(p.Command) == "" {
			continue
		}
		if p.ID == "workbench-runner" && strings.TrimSpace(prefs.OpenClawSSHHost) == "" {
			continue
		}
		if p.ID == "openclaw" && strings.TrimSpace(p.Command) == "" && strings.TrimSpace(prefs.OpenClawCommand) == "" && strings.TrimSpace(prefs.OpenClawSSHHost) == "" {
			continue
		}
		if p.Cost == CostMetered && !prefs.AllowMeteredAPI {
			continue
		}
		out = append(out, p)
	}
	if strings.TrimSpace(prefs.OpenClawCommand) != "" || strings.TrimSpace(prefs.OpenClawSSHHost) != "" {
		found := false
		for _, p := range out {
			if p.ID == "openclaw" {
				found = true
			}
		}
		if !found {
			out = append(out, Provider{ID: "openclaw", Name: "OpenClaw / Harness", Capability: "configured harness", Installed: true, Authenticated: true, Cost: CostIncluded, Priority: 50, CanWrite: true, CanRunTools: true})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if prefs.AvoidWorkUsage {
			wa, wb := a.Cost == CostScarce, b.Cost == CostScarce
			if wa != wb {
				return !wa
			}
		}
		return a.Priority < b.Priority
	})
	return out
}

func attemptSummary(res RunResult, err error) string {
	if err == nil {
		return "completed"
	}
	if res.Authentication {
		return "not authenticated / entitlement unavailable"
	}
	if res.Retryable {
		return "unavailable; tried next eligible worker"
	}
	return "failed; tried next eligible worker"
}

func (e *Engine) updateRunning(id string, p Provider) {
	e.mu.Lock()
	i := e.taskIndexLocked(id)
	if i < 0 {
		e.mu.Unlock()
		return
	}
	now := time.Now()
	e.state.Tasks[i].Status = TaskRunning
	e.state.Tasks[i].ProviderID = p.ID
	e.state.Tasks[i].ConsumesWork = p.Cost == CostScarce
	e.state.Tasks[i].RouteReason = routeReason(p)
	e.state.Tasks[i].UpdatedAt = now
	if e.state.Tasks[i].StartedAt == nil {
		e.state.Tasks[i].StartedAt = &now
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
}
func routeReason(p Provider) string {
	switch p.Cost {
	case CostZero:
		return "zero-marginal-cost eligible worker"
	case CostIncluded:
		return "included subscription worker before scarce Work"
	case CostScarce:
		return "escalated to scarce Work/Codex only after cheaper eligible routes"
	default:
		return "explicit metered route"
	}
}
func (e *Engine) appendAttempt(id, s string) {
	e.mu.Lock()
	i := e.taskIndexLocked(id)
	if i >= 0 {
		e.state.Tasks[i].Attempts = append(e.state.Tasks[i].Attempts, s)
		e.state.Tasks[i].UpdatedAt = time.Now()
	}
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
}
func (e *Engine) finishCompleted(id string, p Provider, res RunResult) {
	e.mu.Lock()
	i := e.taskIndexLocked(id)
	if i < 0 {
		e.mu.Unlock()
		return
	}
	now := time.Now()
	e.state.Tasks[i].Status = TaskCompleted
	e.state.Tasks[i].ProviderID = p.ID
	e.state.Tasks[i].Output = strings.TrimSpace(res.Output)
	e.state.Tasks[i].Review = cloneTaskReviewResult(res.Review)
	e.state.Tasks[i].Error = ""
	e.state.Tasks[i].UpdatedAt = now
	e.state.Tasks[i].FinishedAt = &now
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
}
func (e *Engine) finishAttention(id string, p Provider, res RunResult) {
	e.mu.Lock()
	i := e.taskIndexLocked(id)
	if i < 0 {
		e.mu.Unlock()
		return
	}
	e.state.Tasks[i].Status = TaskNeedsAttention
	e.state.Tasks[i].ProviderID = p.ID
	e.state.Tasks[i].Output = res.Output
	e.state.Tasks[i].AttentionQuestion = res.Attention
	e.state.Tasks[i].UpdatedAt = time.Now()
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
}
func (e *Engine) finishFailed(id, msg string) {
	e.mu.Lock()
	i := e.taskIndexLocked(id)
	if i < 0 {
		e.mu.Unlock()
		return
	}
	now := time.Now()
	e.state.Tasks[i].Status = TaskFailed
	e.state.Tasks[i].Error = msg
	e.state.Tasks[i].UpdatedAt = now
	e.state.Tasks[i].FinishedAt = &now
	st := cloneState(e.state)
	e.mu.Unlock()
	_ = e.store.Save(st)
	e.notify()
}
func (e *Engine) taskIndexLocked(id string) int {
	for i := range e.state.Tasks {
		if e.state.Tasks[i].ID == id {
			return i
		}
	}
	return -1
}

func (e *Engine) sendAttentionNotification(taskID, question, template string) {
	if strings.TrimSpace(template) == "" {
		return
	}
	msg := fmt.Sprintf("Workbench needs you: %s (task %s)", question, taskID)
	line := strings.ReplaceAll(template, "{message}", shellQuote(msg))
	var cmd *exec.Cmd
	if isWindows() {
		cmd = exec.Command("cmd.exe", "/D", "/S", "/C", line)
	} else {
		cmd = exec.Command("/bin/sh", "-lc", line)
	}
	configureChildProcess(cmd, false)
	_ = cmd.Start()
}