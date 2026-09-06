package runnerdiag

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunnerDiagnosticSettingsProjection(t *testing.T) {
	raw := []byte(`{"agentId":123,"gitHubUrl":"https://github.com/example/project","workFolder":"_work","agentName":"private-name","unknown":"PRIVATE_SENTINEL"}`)
	s, target, e := decodeSettings(raw)
	if e != nil || s.AgentID != 123 || target != "https://github.com/example/project" || workFolderClass(s.WorkFolder) != "relative_not_inspected" {
		t.Fatal("projection failed")
	}
	r := newReport()
	r.Installations = append(r.Installations, Installation{AgentID: s.AgentID, Target: target})
	out, e := encodeReport(r)
	if e != nil || strings.Contains(out, "PRIVATE_SENTINEL") || strings.Contains(out, "private-name") {
		t.Fatal("raw configuration leaked")
	}
	if r.MachineWideAbsenceEstablished || r.UnrealProof != "not_executed" || !r.ReadOnly {
		t.Fatal("false proof")
	}
}
func TestRunnerDiagnosticRejectsUnsafeSettings(t *testing.T) {
	cases := []string{`null`, `[]`, `{}`, `{"agentId":0}`, `{"agentId":1,"AgentID":2,"gitHubUrl":"https://github.com/example/repo"}`, `{"agentId":1,"gitHubUrl":"https://github.com/example/repo"} {}`,
		`{"agentId":1,"gitHubUrl":"https://token@github.com/example/repo"}`, `{"agentId":1,"gitHubUrl":"https://github.com/example/repo?token=private"}`, `{"agentId":1,"gitHubUrl":"file:///private"}`, `{"agentId":1,"gitHubUrl":"https://github.com/../repo"}`, strings.Repeat("a", MaxConfigBytes+1)}
	for _, raw := range cases {
		if _, _, e := decodeSettings([]byte(raw)); e == nil {
			t.Fatal("accepted unsafe metadata")
		}
	}
}
func TestRunnerDiagnosticTargets(t *testing.T) {
	for _, value := range []string{"https://github.com/example", "https://github.com/example/repo/"} {
		if _, ok := githubTarget(value); !ok {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"https://github.com.evil/a", "https://github.com:443/a", "https://github.com/a/b/c", "https://github.com/a%2fb", "https://github.com/a#secret", "https://github.com/a?", "http://github.com/a", "https://github.com/."} {
		if _, ok := githubTarget(value); ok {
			t.Fatal(value)
		}
	}
}
func TestRunnerDiagnosticWorkFolderNotFollowed(t *testing.T) {
	for _, value := range []string{`C:\private`, `..\private`, `\\server\private`, `.git`, `a/../b`, "/tmp", "."} {
		if workFolderClass(value) != "external_or_unsafe_not_inspected" {
			t.Fatal(value)
		}
	}
}
func TestRunnerDiagnosticRegistrationEvidence(t *testing.T) {
	i := Installation{Config: "readable", AgentID: 123, Target: "https://github.com/example/repo"}
	cases := []struct {
		scope    string
		complete bool
		ids      []uint64
		want     string
	}{
		{i.Target, true, []uint64{123}, "registration_present_availability_not_assessed"},
		{i.Target, true, []uint64{}, "local_configuration_orphaned_in_observed_scope"},
		{i.Target, false, nil, "registration_unknown"},
		{"https://github.com/example", true, []uint64{123}, "registration_unknown"},
		{"https://github.com/other/repo", true, nil, "registration_unknown"},
	}
	for _, c := range cases {
		if got := ClassifyRegistration(i, c.scope, c.complete, c.ids); got != c.want {
			t.Fatalf("%s != %s", got, c.want)
		}
	}
	i.Config = "absent"
	if got := ClassifyRegistration(i, i.Target, true, []uint64{123}); got != "local_registration_not_established" {
		t.Fatal(got)
	}
}
func TestRunnerDiagnosticOutputBoundAndNoPathFields(t *testing.T) {
	r := newReport()
	r.ServicesRead = strings.Repeat("x", MaxOutputBytes)
	if _, e := encodeReport(r); e == nil {
		t.Fatal("oversized report accepted")
	}
	raw, _ := encodeReport(newReport())
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil || m["machine_wide_absence_established"] != false {
		t.Fatal("false absence")
	}
}
