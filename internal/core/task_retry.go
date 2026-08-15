package core

import (
	"fmt"
	"strings"
	"time"
)

const maxAutomaticTaskRetries = 3

func automaticProviderRetryEligible(provider Provider, record ProviderHealth) bool {
	if provider.Cost == CostScarce || provider.Cost == CostMetered || !record.CooldownUntil.After(time.Now().Add(-time.Second)) {
		return false
	}
	switch strings.TrimSpace(record.Reason) {
	case "quota or rate limit", "worker timed out", "worker temporarily unavailable":
		return true
	default:
		return false
	}
}

func earlierAutomaticRetry(current time.Time, provider Provider, record ProviderHealth) time.Time {
	if !automaticProviderRetryEligible(provider, record) {
		return current
	}
	candidate := record.CooldownUntil.UTC()
	if current.IsZero() || candidate.Before(current) {
		return candidate
	}
	return current
}

func automaticRetryAtForCoolingProviders(providers []Provider, now time.Time) (time.Time, bool) {
	st, err := loadProviderHealthState()
	if err != nil {
		return time.Time{}, false
	}
	active := activeProviderHealth(st, now)
	var retryAt time.Time
	for _, provider := range providers {
		record, ok := active[provider.ID]
		if !ok {
			continue
		}
		retryAt = earlierAutomaticRetry(retryAt, provider, record)
	}
	return retryAt, !retryAt.IsZero()
}

func (e *Engine) deferAutomaticRetry(taskID string, retryAt time.Time) (bool, error) {
	retryAt = retryAt.UTC()
	now := time.Now().UTC()
	if retryAt.Before(now.Add(time.Second)) {
		retryAt = now.Add(time.Second)
	}

	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 {
		e.mu.Unlock()
		return false, nil
	}
	if e.state.Tasks[i].AutoRetryCount >= maxAutomaticTaskRetries {
		e.mu.Unlock()
		return false, nil
	}
	e.state.Tasks[i].AutoRetryCount++
	count := e.state.Tasks[i].AutoRetryCount
	e.state.Tasks[i].Status = TaskWaitingRetry
	e.state.Tasks[i].ProviderID = ""
	e.state.Tasks[i].RouteReason = "waiting for a transient provider cooldown"
	e.state.Tasks[i].ConsumesWork = false
	e.state.Tasks[i].RetryAt = &retryAt
	e.state.Tasks[i].FinishedAt = nil
	e.state.Tasks[i].UpdatedAt = now
	e.state.Tasks[i].Attempts = append(e.state.Tasks[i].Attempts,
		fmt.Sprintf("Workbench: transient provider availability; automatic retry %d/%d scheduled for %s", count, maxAutomaticTaskRetries, retryAt.Format(time.RFC3339)))
	st := cloneState(e.state)
	e.mu.Unlock()

	if err := e.store.Save(st); err != nil {
		return false, err
	}
	e.notify()
	go e.scheduleAutomaticRetry(taskID, retryAt)
	return true, nil
}

func (e *Engine) scheduleAutomaticRetry(taskID string, retryAt time.Time) {
	delay := time.Until(retryAt)
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
	}

	e.mu.Lock()
	i := e.taskIndexLocked(taskID)
	if i < 0 || e.state.Tasks[i].Status != TaskWaitingRetry || e.state.Tasks[i].RetryAt == nil || !e.state.Tasks[i].RetryAt.Equal(retryAt) {
		e.mu.Unlock()
		return
	}
	now := time.Now().UTC()
	e.state.Tasks[i].Status = TaskQueued
	e.state.Tasks[i].ProviderID = ""
	e.state.Tasks[i].RouteReason = ""
	e.state.Tasks[i].ConsumesWork = false
	e.state.Tasks[i].RetryAt = nil
	e.state.Tasks[i].UpdatedAt = now
	e.state.Tasks[i].Attempts = append(e.state.Tasks[i].Attempts, "Workbench: provider cooldown expired; retrying automatically")
	st := cloneState(e.state)
	e.mu.Unlock()

	if err := e.store.Save(st); err != nil {
		e.finishFailed(taskID, "Workbench could not persist the scheduled automatic retry: "+err.Error())
		return
	}
	e.notify()
	go e.execute(taskID)
}
