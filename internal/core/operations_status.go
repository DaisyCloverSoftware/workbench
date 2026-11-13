package core

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// OperationsStatus is executor-owned lifecycle evidence. It never contains
// subprocess output. Report, when present, is explicitly a script assertion,
// not independently observed application state or proof of message delivery.
// Only newly created per-execution reporting files are accepted; no historical
// output, log or audit file is read by this contract.
type OperationsStatus struct {
	SourceCommit     string                 `json:"source_commit,omitempty"`
	ScriptSHA256     string                 `json:"script_sha256,omitempty"`
	Version          int                    `json:"version"`
	ExecutorVersion  string                 `json:"executor_version"`
	Started          bool                   `json:"started"`
	Completion       string                 `json:"completion"`
	ProcessExitCode  *int                   `json:"process_exit_code"`
	ProcessObserved  bool                   `json:"process_result_observed"`
	OutputWithheld   bool                   `json:"output_withheld"`
	OutputTruncated  bool                   `json:"output_truncated"`
	EvidenceComplete bool                   `json:"evidence_complete"`
	Reporting        string                 `json:"reporting"`
	Report           *OperationsDisposition `json:"script_reported_disposition,omitempty"`
}

// OperationsDisposition has no free-text slots, paths, identifiers, collections
// or raw errors. Every field is mandatory, checked against a closed vocabulary,
// and reserialised from these typed values. It cannot expose arbitrary text from
// a protected stream. Identity comes from the executor/request, never this file.
type OperationsDisposition struct {
	Version        int    `json:"version"`
	Terminal       bool   `json:"terminal"`
	Stage          string `json:"stage"`
	Reason         string `json:"reason"`
	Update         string `json:"update"`
	Recovery       string `json:"recovery"`
	Authentication string `json:"authentication"`
	Health         string `json:"health"`
	Readiness      string `json:"readiness"`
	Audit          string `json:"audit"`
}

const maxOperationsDispositionBytes = 2048

func newOperationsStatus() *OperationsStatus {
	return &OperationsStatus{Version: 1, ExecutorVersion: Version,
		Completion: "not_started", Reporting: "not_reported"}
}

func operationsEnum(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

// decodeOperationsDisposition rejects duplicate, unknown, missing, null, nested,
// oversized and trailing data. It returns no input bytes or decoder diagnostics.
func decodeOperationsDisposition(body []byte) (*OperationsDisposition, bool) {
	if len(body) == 0 || len(body) > maxOperationsDispositionBytes {
		return nil, false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, false
	}
	fields := make(map[string]json.RawMessage, 10)
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok || !operationsEnum(key, "version", "terminal", "stage", "reason", "update", "recovery", "authentication", "health", "readiness", "audit") {
			return nil, false
		}
		if _, duplicate := fields[key]; duplicate {
			return nil, false
		}
		var value json.RawMessage
		if d.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, false
		}
		fields[key] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') || len(fields) != 10 {
		return nil, false
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return nil, false
	}
	var report OperationsDisposition
	if json.Unmarshal(body, &report) != nil || report.Version != 1 {
		return nil, false
	}
	if !operationsEnum(report.Stage, "unknown", "initialisation", "preflight", "update", "readback", "restart", "pod_consistency", "authentication", "health", "readiness", "startup_logs", "preservation", "source_readback", "rollback_precheck", "rollback_update", "rollback_readback", "rollback_restart", "rollback_verification", "complete") ||
		!operationsEnum(report.Reason, "none", "unknown", "management", "baseline", "configuration", "concurrency", "command_failed", "timeout", "interrupted", "invalid_response", "pod_changed", "authentication_rejected", "tls_failed", "connection_failed", "provider_rejected", "health_failed", "readiness_failed", "startup_log_missing", "startup_log_incomplete", "preservation_failed", "audit_failed", "reporting_failed", "internal_failure") ||
		!operationsEnum(report.Update, "unknown", "not_attempted", "not_applied", "confirmed", "unexpected") ||
		!operationsEnum(report.Recovery, "unknown", "not_attempted", "confirmed", "unconfirmed", "blocked_concurrent_change", "not_attempted_update_unresolved") ||
		!operationsEnum(report.Authentication, "unknown", "passed", "failed", "not_attempted") ||
		!operationsEnum(report.Health, "unknown", "200", "503", "other", "not_attempted") ||
		!operationsEnum(report.Readiness, "unknown", "200", "503", "other", "not_attempted") ||
		!operationsEnum(report.Audit, "unknown", "complete", "incomplete") {
		return nil, false
	}
	return &report, true
}

// readOperationsDisposition reads only the handle created before this execution.
// A script cannot redirect the reader to a symlink, FIFO, old audit or other file.
func readOperationsDisposition(file *os.File, identity os.FileInfo) (*OperationsDisposition, string) {
	if file == nil || identity == nil {
		return nil, "unavailable"
	}
	now, err := os.Lstat(file.Name())
	if err != nil || !now.Mode().IsRegular() || !os.SameFile(identity, now) || now.Mode().Perm() != 0o600 {
		return nil, "invalid"
	}
	if now.Size() == 0 {
		return nil, "not_reported"
	}
	if now.Size() > maxOperationsDispositionBytes {
		return nil, "invalid"
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, "unavailable"
	}
	body, err := io.ReadAll(io.LimitReader(file, maxOperationsDispositionBytes+1))
	if err != nil {
		return nil, "unavailable"
	}
	after, statErr := file.Stat()
	pathAfter, pathErr := os.Lstat(file.Name())
	if statErr != nil || pathErr != nil || !os.SameFile(identity, pathAfter) || !os.SameFile(now, after) || after.Size() != int64(len(body)) || !after.ModTime().Equal(now.ModTime()) {
		return nil, "invalid"
	}
	report, ok := decodeOperationsDisposition(body)
	if !ok {
		return nil, "invalid"
	}
	if !report.Terminal || report.Audit != "complete" {
		return report, "incomplete"
	}
	return report, "complete"
}
