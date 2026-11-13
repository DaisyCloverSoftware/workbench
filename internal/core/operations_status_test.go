package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testDisposition = `{"version":1,"terminal":true,"stage":"complete","reason":"none","update":"confirmed","recovery":"not_attempted","authentication":"passed","health":"200","readiness":"200","audit":"complete"}`

func TestOperationsDispositionClosedContract(t *testing.T) {
	r, ok := decodeOperationsDisposition([]byte(testDisposition))
	if !ok || r.Update != "confirmed" || !r.Terminal {
		t.Fatal("valid fixed report rejected")
	}
	for _, bad := range []string{
		"", "null", "[]", testDisposition + testDisposition,
		strings.Replace(testDisposition, `"version":1`, `"version":2`, 1),
		strings.Replace(testDisposition, `"version":1`, `"version":1,"version":1`, 1),
		strings.Replace(testDisposition, `"terminal":true,`, "", 1),
		strings.Replace(testDisposition, `"terminal":true`, `"terminal":null`, 1),
		strings.Replace(testDisposition, `"terminal":true`, `"terminal":"true"`, 1),
		strings.Replace(testDisposition, `"reason":"none"`, `"reason":"dummy-private-diagnostic"`, 1),
		strings.Replace(testDisposition, `"audit":"complete"`, `"audit":"complete","raw_error":"dummy-private-diagnostic"`, 1),
		strings.Replace(testDisposition, `"health":"200"`, `"health":{"unexpected":"dummy"}`, 1),
		strings.Replace(testDisposition, `"stage":"complete"`, `"stage":"`+strings.Repeat("x", 2048)+`"`, 1),
		strings.Replace(testDisposition, `"reason":"none"`, `"reason":"none","started":true`, 1),
	} {
		if r, accepted := decodeOperationsDisposition([]byte(bad)); accepted || r != nil {
			t.Fatal("invalid report accepted")
		}
	}
	// All non-enum payloads, including credential-like values, are rejected as
	// data, not scrubbed/renamed to slip through output protection.
	for _, field := range []string{"stage", "reason", "update", "recovery", "authentication", "health", "readiness", "audit"} {
		var m map[string]any
		_ = json.Unmarshal([]byte(testDisposition), &m)
		m[field] = "dummy-non-enumerated-credential-value"
		b, _ := json.Marshal(m)
		if _, ok := decodeOperationsDisposition(b); ok {
			t.Fatalf("free text allowed in %s", field)
		}
	}
}

func TestOperationsDispositionFreshFileOnly(t *testing.T) {
	file, err := os.OpenFile(filepath.Join(t.TempDir(), "status"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	info, _ := file.Stat()
	if report, state := readOperationsDisposition(file, info); report != nil || state != "not_reported" {
		t.Fatal(state)
	}
	if _, err = file.WriteString(testDisposition); err != nil {
		t.Fatal(err)
	}
	if report, state := readOperationsDisposition(file, info); report == nil || state != "complete" {
		t.Fatal(state)
	}
	_ = file.Truncate(0)
	_, _ = file.Seek(0, 0)
	_, _ = file.WriteString(strings.Replace(testDisposition, `"audit":"complete"`, `"audit":"incomplete"`, 1))
	if report, state := readOperationsDisposition(file, info); report == nil || state != "incomplete" {
		t.Fatal(state)
	}
	// Never follow a replacement inode or a symlink to any previous result.
	old := file.Name() + ".moved"
	if err := os.Rename(file.Name(), old); err != nil {
		t.Skip("platform disallows rename of open fixture")
	}
	if err := os.WriteFile(file.Name(), []byte(testDisposition), 0600); err != nil {
		t.Fatal(err)
	}
	if report, state := readOperationsDisposition(file, info); report != nil || state != "invalid" {
		t.Fatal(state)
	}
	_ = os.Remove(file.Name())
	if err := os.Symlink(old, file.Name()); err == nil {
		if report, state := readOperationsDisposition(file, info); report != nil || state != "invalid" {
			t.Fatal(state)
		}
	}
}

func TestOperationsDispositionIncompleteAndInvalidAreNotSuccess(t *testing.T) {
	for _, tc := range []struct{ body, state string }{
		{strings.Replace(testDisposition, `"terminal":true`, `"terminal":false`, 1), "incomplete"},
		{strings.Replace(testDisposition, `"audit":"complete"`, `"audit":"unknown"`, 1), "incomplete"},
		{"{", "invalid"},
		{strings.Repeat("x", 2049), "invalid"},
	} {
		f, e := os.OpenFile(filepath.Join(t.TempDir(), "status"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if e != nil {
			t.Fatal(e)
		}
		info, _ := f.Stat()
		_, _ = f.WriteString(tc.body)
		_, state := readOperationsDisposition(f, info)
		_ = f.Close()
		if state != tc.state {
			t.Fatalf("expected %s got %s", tc.state, state)
		}
	}
}
