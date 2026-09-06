// Package runnerdiag implements a bounded local observation, never runner recovery.
package runnerdiag

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

const MaxInstallations = 8
const MaxRecords = 32
const MaxConfigBytes = 32 << 10
const MaxOutputBytes = 14 << 10

// Paths, credentials, command lines and service accounts are not report fields.
type Report struct {
	Schema                        int            `json:"schema"`
	ObservedAt                    string         `json:"observed_at_utc"`
	ReadOnly                      bool           `json:"read_only"`
	ServicesRead                  string         `json:"services_read"`
	ProcessesRead                 string         `json:"processes_read"`
	CandidateLimitReached         bool           `json:"candidate_limit_reached"`
	LocationsChecked              int            `json:"locations_checked"`
	LocationsAbsent               int            `json:"locations_absent"`
	LocationsUnreadable           int            `json:"locations_unreadable"`
	Locations                     []Location     `json:"locations"`
	Installations                 []Installation `json:"installations"`
	Services                      []Service      `json:"services"`
	Processes                     []Process      `json:"processes"`
	MachineWideAbsenceEstablished bool           `json:"machine_wide_absence_established"`
	GitHubRegistration            string         `json:"github_registration"`
	LabelsObservation             string         `json:"labels_observation"`
	UnrealProof                   string         `json:"unreal_proof"`
}
type Location struct {
	ID         string `json:"location_id"`
	Source     string `json:"source"`
	ReadStatus string `json:"read_status"`
}
type Installation struct {
	ID            string `json:"installation_id"`
	Source        string `json:"discovery_source"`
	Config        string `json:"runner_config"`
	ServiceMarker string `json:"service_marker"`
	Listener      string `json:"listener_binary"`
	ServiceBinary string `json:"service_binary"`
	ConfigCommand string `json:"config_command"`
	AgentID       uint64 `json:"local_agent_id,omitempty"`
	Target        string `json:"github_target,omitempty"`
	WorkFolder    string `json:"work_folder_class"`
	Ephemeral     bool   `json:"ephemeral"`
	DisableUpdate bool   `json:"disable_update"`
	LocalState    string `json:"local_state"`
}
type Service struct {
	ID             string `json:"service_id"`
	State          string `json:"state"`
	StartMode      string `json:"start_mode"`
	ConfigRead     string `json:"config_read"`
	InstallationID string `json:"installation_id,omitempty"`
}
type Process struct {
	Kind           string `json:"kind"`
	ImageRead      string `json:"image_read"`
	InstallationID string `json:"installation_id,omitempty"`
}
type runnerSettings struct {
	AgentID       uint64 `json:"agentId"`
	GitHubURL     string `json:"gitHubUrl"`
	WorkFolder    string `json:"workFolder"`
	Ephemeral     bool   `json:"ephemeral"`
	DisableUpdate bool   `json:"disableUpdate"`
}

var slug = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,99}$`)

func newReport() Report {
	return Report{Schema: 1, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), ReadOnly: true,
		ServicesRead: "not_observed", ProcessesRead: "not_observed", Locations: []Location{}, Installations: []Installation{},
		Services: []Service{}, Processes: []Process{}, GitHubRegistration: "unverified_requires_fresh_matching_scope_inventory",
		LabelsObservation: "not_in_standard_runner_settings_use_github_inventory", UnrealProof: "not_executed"}
}
func opaqueID(kind, value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(value)))
	return kind + "_" + hex.EncodeToString(sum[:12])
}
func githubTarget(value string) (string, bool) {
	if len(value) > 240 {
		return "", false
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" {
		return "", false
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), "/"), "/")
	if len(parts) < 1 || len(parts) > 2 {
		return "", false
	}
	for _, part := range parts {
		if !slug.MatchString(part) || part == "." || part == ".." {
			return "", false
		}
	}
	return "https://github.com/" + strings.Join(parts, "/"), true
}
func workFolderClass(value string) string {
	// Do not follow this path or include it in the report.
	p := strings.ReplaceAll(value, `\`, "/")
	if p == "" {
		return "not_recorded"
	}
	if strings.HasPrefix(p, "/") || strings.ContainsAny(p, ":\x00\r\n") {
		return "external_or_unsafe_not_inspected"
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." || strings.EqualFold(part, ".git") {
			return "external_or_unsafe_not_inspected"
		}
	}
	if path.Clean(p) == "." {
		return "external_or_unsafe_not_inspected"
	}
	return "relative_not_inspected"
}
func decodeSettings(raw []byte) (runnerSettings, string, error) {
	var s runnerSettings
	if len(raw) == 0 || len(raw) > MaxConfigBytes {
		return s, "", errors.New("invalid_size")
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	// Reject case-insensitive duplicate JSON keys rather than accept last-wins IDs.
	d := json.NewDecoder(bytes.NewReader(raw))
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return s, "", errors.New("invalid_object")
	}
	seen := map[string]bool{}
	for d.More() {
		key, e := d.Token()
		if e != nil {
			return s, "", errors.New("invalid_json")
		}
		name, ok := key.(string)
		if !ok || seen[strings.ToLower(name)] {
			return s, "", errors.New("duplicate_key")
		}
		seen[strings.ToLower(name)] = true
		var discard json.RawMessage
		if d.Decode(&discard) != nil {
			return s, "", errors.New("invalid_json")
		}
	}
	if _, err = d.Token(); err != nil {
		return s, "", errors.New("invalid_json")
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return s, "", errors.New("trailing_json")
	}
	if json.Unmarshal(raw, &s) != nil || s.AgentID == 0 {
		return runnerSettings{}, "", errors.New("invalid_settings")
	}
	target, ok := githubTarget(s.GitHubURL)
	if !ok {
		return runnerSettings{}, "", errors.New("target_withheld_invalid_or_non_github")
	}
	return s, target, nil
}
func encodeReport(r Report) (string, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return "", errors.New("report_encoding_failed")
	}
	if len(raw) > MaxOutputBytes {
		return "", errors.New("report_size_limit")
	}
	return string(raw), nil
}

// ClassifyRegistration compares only a matching, freshly read GitHub scope.
// Callers must not mark complete when the scope or time coverage is unknown.
func ClassifyRegistration(i Installation, scope string, complete bool, ids []uint64) string {
	if i.Config != "readable" || i.AgentID == 0 || i.Target == "" {
		return "local_registration_not_established"
	}
	target, ok := githubTarget(scope)
	if !ok || !strings.EqualFold(target, i.Target) || !complete {
		return "registration_unknown"
	}
	for _, id := range ids {
		if id == i.AgentID {
			return "registration_present_availability_not_assessed"
		}
	}
	return "local_configuration_orphaned_in_observed_scope"
}
