package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	HarnessProtocolVersion = 1
	maxHarnessJobBytes      = 512 << 10
)

type HarnessJobStatus string

const (
	HarnessJobCompleted      HarnessJobStatus = "completed"
	HarnessJobNeedsAttention HarnessJobStatus = "needs_attention"
	HarnessJobUnavailable    HarnessJobStatus = "unavailable"
	HarnessJobFailed         HarnessJobStatus = "failed"
)

type HarnessFailureCategory string

const (
	HarnessFailureAuthentication HarnessFailureCategory = "authentication"
	HarnessFailureQuota          HarnessFailureCategory = "quota"
	HarnessFailurePermissions    HarnessFailureCategory = "permissions"
	HarnessFailureTimeout        HarnessFailureCategory = "timeout"
	HarnessFailureTemporary      HarnessFailureCategory = "temporary"
	HarnessFailureAdapter        HarnessFailureCategory = "adapter"
)

// HarnessCapabilities is an explicit least-authority contract. A structured
// adapter is being asked to work only inside the isolated repository it is
// given. Publication, deployment, secret access, and arbitrary network authority
// remain Workbench/operator responsibilities rather than worker capabilities.
type HarnessCapabilities struct {
	RepositoryRead  bool `json:"repository_read"`
	RepositoryWrite bool `json:"repository_write"`
	LocalCommands   bool `json:"local_commands"`
	NetworkAccess   bool `json:"network_access"`
	Publish         bool `json:"publish"`
	Deploy          bool `json:"deploy"`
	Secrets         bool `json:"secrets"`
}

// HarnessJob is the versioned stdin contract for third-party coding harness
// adapters. It deliberately contains no publication target, credentials, vault
// values, remote URL, or provider account information.
type HarnessJob struct {
	Version      int                 `json:"version"`
	TaskID       string              `json:"task_id"`
	ProjectPath  string              `json:"project_path"`
	Intent       string              `json:"intent"`
	Prompt       string              `json:"prompt"`
	Capabilities HarnessCapabilities `json:"capabilities"`
}

// HarnessJobResult is the versioned stdout contract. Review commits and remote
// publication are not adapter outputs: Workbench finalises the isolated
// workspace itself after a successful result.
type HarnessJobResult struct {
	Version     int                    `json:"version"`
	TaskID      string                 `json:"task_id"`
	Status      HarnessJobStatus       `json:"status"`
	Report      string                 `json:"report,omitempty"`
	Attention   string                 `json:"attention,omitempty"`
	Unavailable string                 `json:"unavailable,omitempty"`
	Category    HarnessFailureCategory `json:"category,omitempty"`
	Retryable   bool                   `json:"retryable,omitempty"`
}

func BuildHarnessJob(task Task, prompt string) HarnessJob {
	return HarnessJob{
		Version:     HarnessProtocolVersion,
		TaskID:      strings.TrimSpace(task.ID),
		ProjectPath: strings.TrimSpace(task.ProjectPath),
		Intent:      strings.TrimSpace(task.Intent),
		Prompt:      prompt,
		Capabilities: HarnessCapabilities{
			RepositoryRead:  true,
			RepositoryWrite: true,
			LocalCommands:   true,
			NetworkAccess:   false,
			Publish:         false,
			Deploy:          false,
			Secrets:         false,
		},
	}
}

// RunHarnessAdapter executes one explicitly configured adapter executable
// directly, without a command shell or operator/model-supplied arguments. The
// adapter receives one HarnessJob JSON object on stdin and must return exactly
// one HarnessJobResult JSON object on stdout.
func RunHarnessAdapter(ctx context.Context, adapterPath string, task Task, prompt string) (RunResult, error) {
	path, err := validateHarnessAdapterPath(adapterPath)
	if err != nil {
		return RunResult{Retryable: true}, err
	}
	job := BuildHarnessJob(task, prompt)
	if job.TaskID == "" || job.ProjectPath == "" || job.Intent == "" {
		return RunResult{}, errors.New("structured harness job requires task id, project path, and intent")
	}
	body, err := json.Marshal(job)
	if err != nil {
		return RunResult{}, err
	}
	if len(body) > maxHarnessJobBytes {
		return RunResult{}, fmt.Errorf("structured harness job exceeds %d bytes", maxHarnessJobBytes)
	}
	body = append(body, '\n')

	cmd := exec.CommandContext(ctx, path)
	cmd.Dir = job.ProjectPath
	cmd.Stdin = bytes.NewReader(body)
	configureChildProcess(cmd, false)
	stdout := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	stderr := newBoundedWorkerCapture(maxWorkerStreamCaptureBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if stdout.Truncated() {
		return RunResult{Retryable: true}, errors.New("structured harness response exceeded Workbench's bounded output limit")
	}

	result, decodeErr := decodeHarnessJobResult([]byte(stdout.String()))
	if decodeErr != nil {
		if runErr != nil {
			return RunResult{Retryable: true}, fmt.Errorf("structured harness adapter exited without a valid result: %w", runErr)
		}
		return RunResult{Retryable: true}, decodeErr
	}
	if result.TaskID != job.TaskID {
		return RunResult{Retryable: true}, errors.New("structured harness result task identity mismatch")
	}
	res, resultErr := harnessResultToRunResult(result)
	if runErr != nil && result.Status != HarnessJobFailed && result.Status != HarnessJobUnavailable {
		res.Retryable = true
		return boundRunResultForPersistence(res), fmt.Errorf("structured harness adapter exited unexpectedly: %w", runErr)
	}
	return boundRunResultForPersistence(res), resultErr
}

func validateHarnessAdapterPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("structured harness adapter is not configured")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("structured harness adapter cannot be resolved: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("structured harness adapter must be one regular executable file")
	}
	return resolved, nil
}

func decodeHarnessJobResult(body []byte) (HarnessJobResult, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var result HarnessJobResult
	if err := dec.Decode(&result); err != nil {
		return HarnessJobResult{}, fmt.Errorf("invalid structured harness result: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return HarnessJobResult{}, errors.New("structured harness result contains more than one JSON value")
		}
		return HarnessJobResult{}, fmt.Errorf("invalid trailing structured harness output: %w", err)
	}
	if result.Version != HarnessProtocolVersion {
		return HarnessJobResult{}, fmt.Errorf("unsupported structured harness protocol version %d", result.Version)
	}
	result.TaskID = strings.TrimSpace(result.TaskID)
	if result.TaskID == "" {
		return HarnessJobResult{}, errors.New("structured harness result is missing task_id")
	}
	if !validHarnessFailureCategory(result.Category) {
		return HarnessJobResult{}, fmt.Errorf("unknown structured harness failure category %q", result.Category)
	}
	result.Report = boundPersistedWorkerText(result.Report)
	result.Attention = boundWorkerControlText(result.Attention)
	result.Unavailable = boundWorkerControlText(result.Unavailable)
	return result, nil
}

func validHarnessFailureCategory(category HarnessFailureCategory) bool {
	switch category {
	case "", HarnessFailureAuthentication, HarnessFailureQuota, HarnessFailurePermissions, HarnessFailureTimeout, HarnessFailureTemporary, HarnessFailureAdapter:
		return true
	default:
		return false
	}
}

func harnessResultToRunResult(result HarnessJobResult) (RunResult, error) {
	res := RunResult{Output: result.Report, Retryable: result.Retryable}
	switch result.Status {
	case HarnessJobCompleted:
		if result.Attention != "" || result.Unavailable != "" || result.Category != "" {
			return RunResult{Retryable: true}, errors.New("completed structured harness result contains failure or attention fields")
		}
		res.Retryable = false
		return res, nil
	case HarnessJobNeedsAttention:
		if result.Attention == "" {
			return RunResult{Retryable: true}, errors.New("structured harness requested attention without a question")
		}
		if result.Unavailable != "" || result.Category != "" {
			return RunResult{Retryable: true}, errors.New("structured harness attention result also contains unavailable/failure fields")
		}
		res.Retryable = false
		res.Attention = result.Attention
		return res, nil
	case HarnessJobUnavailable:
		if result.Attention != "" {
			return RunResult{Retryable: true}, errors.New("structured harness unavailable result also contains an attention question")
		}
		if result.Unavailable == "" {
			result.Unavailable = "structured harness is unavailable for this task"
		}
		res.WorkerUnavailable = result.Unavailable
		res.Retryable = true
		res.Authentication = result.Category == HarnessFailureAuthentication
		return res, errors.New(result.Unavailable)
	case HarnessJobFailed:
		if result.Attention != "" || result.Unavailable != "" {
			return RunResult{Retryable: true}, errors.New("failed structured harness result contains attention/unavailable fields")
		}
		return res, errors.New("structured harness reported task failure")
	default:
		return RunResult{Retryable: true}, fmt.Errorf("unknown structured harness status %q", result.Status)
	}
}
