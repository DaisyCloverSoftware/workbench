package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RunnerJobStatus string

const (
	RunnerJobQueued         RunnerJobStatus = "queued"
	RunnerJobRunning        RunnerJobStatus = "running"
	RunnerJobNeedsAttention RunnerJobStatus = "needs_attention"
	RunnerJobCompleted      RunnerJobStatus = "completed"
	RunnerJobFailed         RunnerJobStatus = "failed"
	RunnerJobCancelled      RunnerJobStatus = "cancelled"
)

type RunnerJob struct {
	ID         string          `json:"id"`
	TaskID     string          `json:"task_id"`
	Generation int             `json:"generation"`
	Status     RunnerJobStatus `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	Response   *RunnerResponse `json:"response,omitempty"`
	Error      string          `json:"error,omitempty"`
}

type RunnerJobSubmitResult struct {
	Job    RunnerJob `json:"job"`
	Reused bool      `json:"reused"`
}

type runnerJobRecord struct {
	Job         RunnerJob     `json:"job"`
	Request     RunnerRequest `json:"request"`
	Fingerprint string        `json:"request_fingerprint"`
	PID         int           `json:"pid,omitempty"`
}

type runnerJobLauncher func(jobID string) (int, error)
type runnerJobExecutor func(context.Context, RunnerRequest) RunnerResponse

type runnerRequestIdentity struct {
	TaskID           string `json:"task_id"`
	ProjectPath      string `json:"project_path"`
	Intent           string `json:"intent"`
	HumanAnswer      string `json:"human_answer,omitempty"`
	AvoidWorkUsage   bool   `json:"avoid_work_usage"`
	AllowMeteredAPI  bool   `json:"allow_metered_api"`
}

func runnerJobsRoot() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("WORKBENCH_RUNNER_JOB_ROOT")); configured != "" {
		abs, err := filepath.Abs(configured)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(abs, 0o700); err != nil {
			return "", err
		}
		if err := os.Chmod(abs, 0o700); err != nil {
			return "", err
		}
		return filepath.Clean(abs), nil
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(config, "Workbench", "runner-jobs")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return "", err
	}
	return root, nil
}

func validateRunnerJobID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("runner job id is empty")
	}
	if len(id) > 256 {
		return "", errors.New("runner job id is too long")
	}
	for _, r := range id {
		if r < 0x20 || r == 0x7f {
			return "", errors.New("runner job id contains control characters")
		}
	}
	return id, nil
}

func runnerJobPath(root, id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(root, hex.EncodeToString(sum[:])+".json")
}

// runnerRequestFingerprint deliberately excludes mutable desktop lifecycle
// fields such as Task.Status, ProviderID, Attempts and timestamps. Workbench
// rewrites those fields when recovering after a desktop restart; including them
// would make the same user request look like a different remote job and could
// duplicate coding work. HumanAnswer is part of the identity so a genuine
// attention response becomes the next durable generation.
func runnerRequestFingerprint(req RunnerRequest) (string, error) {
	identity := runnerRequestIdentity{
		TaskID:          strings.TrimSpace(req.Task.ID),
		ProjectPath:     strings.TrimSpace(req.Task.ProjectPath),
		Intent:          strings.TrimSpace(req.Task.Intent),
		HumanAnswer:     strings.TrimSpace(req.Task.HumanAnswer),
		AvoidWorkUsage:  req.AvoidWorkUsage,
		AllowMeteredAPI: req.AllowMeteredAPI,
	}
	b, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func SubmitRunnerJob(req RunnerRequest) (RunnerJobSubmitResult, error) {
	return submitRunnerJob(req, spawnDetachedRunnerJob)
}

func submitRunnerJob(req RunnerRequest, launcher runnerJobLauncher) (RunnerJobSubmitResult, error) {
	id, err := validateRunnerJobID(req.Task.ID)
	if err != nil {
		return RunnerJobSubmitResult{}, err
	}
	if strings.TrimSpace(req.Task.ProjectPath) == "" {
		return RunnerJobSubmitResult{}, errors.New("runner job project path is empty")
	}
	if strings.TrimSpace(req.Task.Intent) == "" {
		return RunnerJobSubmitResult{}, errors.New("runner job intent is empty")
	}
	fingerprint, err := runnerRequestFingerprint(req)
	if err != nil {
		return RunnerJobSubmitResult{}, err
	}
	root, err := runnerJobsRoot()
	if err != nil {
		return RunnerJobSubmitResult{}, err
	}

	var created runnerJobRecord
	var reused *RunnerJob
	err = withRunnerJobsLock(root, func() error {
		record, found, loadErr := loadRunnerJobRecord(root, id)
		if loadErr != nil {
			return loadErr
		}
		if found {
			refreshRunnerJobProcessState(&record)
			if record.Fingerprint == fingerprint {
				job := record.Job
				reused = &job
				return saveRunnerJobRecord(root, record)
			}
			if runnerJobActive(record.Job.Status) {
				return errors.New("runner job is already active for this task; wait or cancel it before submitting a different request")
			}
			created.Job.Generation = record.Job.Generation + 1
		}
		if created.Job.Generation == 0 {
			created.Job.Generation = 1
		}
		now := time.Now().UTC()
		created.Job.ID = id
		created.Job.TaskID = id
		created.Job.Status = RunnerJobQueued
		created.Job.CreatedAt = now
		created.Job.UpdatedAt = now
		created.Request = req
		created.Fingerprint = fingerprint
		return saveRunnerJobRecord(root, created)
	})
	if err != nil {
		return RunnerJobSubmitResult{}, err
	}
	if reused != nil {
		return RunnerJobSubmitResult{Job: *reused, Reused: true}, nil
	}

	pid, launchErr := launcher(id)
	if launchErr != nil {
		_ = withRunnerJobsLock(root, func() error {
			record, found, _ := loadRunnerJobRecord(root, id)
			if !found || record.Fingerprint != fingerprint {
				return nil
			}
			now := time.Now().UTC()
			record.Job.Status = RunnerJobFailed
			record.Job.Error = "could not start durable runner worker: " + launchErr.Error()
			record.Job.UpdatedAt = now
			record.Job.FinishedAt = &now
			return saveRunnerJobRecord(root, record)
		})
		return RunnerJobSubmitResult{}, fmt.Errorf("start durable runner job: %w", launchErr)
	}

	_ = withRunnerJobsLock(root, func() error {
		record, found, loadErr := loadRunnerJobRecord(root, id)
		if loadErr != nil || !found || record.Fingerprint != fingerprint {
			return loadErr
		}
		// A very fast worker may already have recorded its terminal result before
		// the submitting process gets here. Never put a stale launcher PID back
		// onto a terminal record.
		if runnerJobActive(record.Job.Status) && record.PID == 0 {
			record.PID = pid
			record.Job.UpdatedAt = time.Now().UTC()
			if err := saveRunnerJobRecord(root, record); err != nil {
				return err
			}
		}
		created = record
		return nil
	})
	return RunnerJobSubmitResult{Job: created.Job}, nil
}

func GetRunnerJob(id string) (RunnerJob, error) {
	id, err := validateRunnerJobID(id)
	if err != nil {
		return RunnerJob{}, err
	}
	root, err := runnerJobsRoot()
	if err != nil {
		return RunnerJob{}, err
	}
	var job RunnerJob
	err = withRunnerJobsLock(root, func() error {
		record, found, loadErr := loadRunnerJobRecord(root, id)
		if loadErr != nil {
			return loadErr
		}
		if !found {
			return errors.New("runner job not found")
		}
		changed := refreshRunnerJobProcessState(&record)
		if changed {
			if err := saveRunnerJobRecord(root, record); err != nil {
				return err
			}
		}
		job = record.Job
		return nil
	})
	return job, err
}

func CancelRunnerJob(id string) (RunnerJob, error) {
	id, err := validateRunnerJobID(id)
	if err != nil {
		return RunnerJob{}, err
	}
	root, err := runnerJobsRoot()
	if err != nil {
		return RunnerJob{}, err
	}
	var job RunnerJob
	var pid int
	err = withRunnerJobsLock(root, func() error {
		record, found, loadErr := loadRunnerJobRecord(root, id)
		if loadErr != nil {
			return loadErr
		}
		if !found {
			return errors.New("runner job not found")
		}
		refreshRunnerJobProcessState(&record)
		if !runnerJobActive(record.Job.Status) {
			job = record.Job
			return saveRunnerJobRecord(root, record)
		}
		now := time.Now().UTC()
		record.Job.Status = RunnerJobCancelled
		record.Job.Error = ""
		record.Job.UpdatedAt = now
		record.Job.FinishedAt = &now
		pid = record.PID
		job = record.Job
		return saveRunnerJobRecord(root, record)
	})
	if err != nil {
		return RunnerJob{}, err
	}
	if pid > 0 {
		_ = terminateRunnerJobProcess(pid)
	}
	return job, nil
}

func ExecuteStoredRunnerJob(id string) error {
	return executeStoredRunnerJob(id, ExecuteRunnerRequest)
}

func executeStoredRunnerJob(id string, executor runnerJobExecutor) error {
	id, err := validateRunnerJobID(id)
	if err != nil {
		return err
	}
	root, err := runnerJobsRoot()
	if err != nil {
		return err
	}
	var req RunnerRequest
	var fingerprint string
	err = withRunnerJobsLock(root, func() error {
		record, found, loadErr := loadRunnerJobRecord(root, id)
		if loadErr != nil {
			return loadErr
		}
		if !found {
			return errors.New("runner job not found")
		}
		if !runnerJobActive(record.Job.Status) {
			return nil
		}
		now := time.Now().UTC()
		record.Job.Status = RunnerJobRunning
		record.Job.UpdatedAt = now
		if record.Job.StartedAt == nil {
			record.Job.StartedAt = &now
		}
		record.PID = os.Getpid()
		req = record.Request
		fingerprint = record.Fingerprint
		return saveRunnerJobRecord(root, record)
	})
	if err != nil {
		return err
	}
	if fingerprint == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	response := executor(ctx, req)

	return withRunnerJobsLock(root, func() error {
		record, found, loadErr := loadRunnerJobRecord(root, id)
		if loadErr != nil {
			return loadErr
		}
		if !found || record.Fingerprint != fingerprint {
			return nil
		}
		if record.Job.Status == RunnerJobCancelled {
			return nil
		}
		now := time.Now().UTC()
		record.PID = 0
		record.Job.Response = &response
		record.Job.Error = strings.TrimSpace(response.Error)
		record.Job.UpdatedAt = now
		record.Job.FinishedAt = &now
		switch {
		case strings.TrimSpace(response.Result.Attention) != "":
			record.Job.Status = RunnerJobNeedsAttention
		case strings.TrimSpace(response.Error) != "":
			record.Job.Status = RunnerJobFailed
		default:
			record.Job.Status = RunnerJobCompleted
		}
		return saveRunnerJobRecord(root, record)
	})
}

func runnerJobActive(status RunnerJobStatus) bool {
	return status == RunnerJobQueued || status == RunnerJobRunning
}

func refreshRunnerJobProcessState(record *runnerJobRecord) bool {
	if record == nil || !runnerJobActive(record.Job.Status) {
		return false
	}
	if record.PID > 0 {
		if runnerJobProcessAlive(record.PID) {
			return false
		}
		now := time.Now().UTC()
		record.PID = 0
		record.Job.Status = RunnerJobFailed
		record.Job.Error = "durable runner worker exited before recording a terminal result"
		record.Job.UpdatedAt = now
		record.Job.FinishedAt = &now
		return true
	}
	if time.Since(record.Job.CreatedAt) > 20*time.Second {
		now := time.Now().UTC()
		record.Job.Status = RunnerJobFailed
		record.Job.Error = "durable runner worker did not start"
		record.Job.UpdatedAt = now
		record.Job.FinishedAt = &now
		return true
	}
	return false
}

func loadRunnerJobRecord(root, id string) (runnerJobRecord, bool, error) {
	path := runnerJobPath(root, id)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return runnerJobRecord{}, false, nil
	}
	if err != nil {
		return runnerJobRecord{}, false, err
	}
	var record runnerJobRecord
	if err := json.Unmarshal(b, &record); err != nil {
		return runnerJobRecord{}, false, fmt.Errorf("decode runner job: %w", err)
	}
	if record.Job.ID != id {
		return runnerJobRecord{}, false, errors.New("runner job record identity mismatch")
	}
	return record, true, nil
}

func saveRunnerJobRecord(root string, record runnerJobRecord) error {
	b, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	path := runnerJobPath(root, record.Job.ID)
	f, err := os.CreateTemp(root, ".runner-job-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func withRunnerJobsLock(root string, fn func() error) error {
	lockPath := filepath.Join(root, ".runner-jobs.lock")
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			defer os.Remove(lockPath)
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 2*time.Minute {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return errors.New("runner job store is busy")
		}
		time.Sleep(25 * time.Millisecond)
	}
}
