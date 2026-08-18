package core

import (
	"archive/tar"
	"bytes"
	"testing"
)

func relayActivityArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range files {
		b := []byte(content)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(b))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseRunnerChatActivityCombinesSafeHandsAndAutonomousWork(t *testing.T) {
	raw := relayActivityArchive(t, map[string]string{
		"relay/control/read_12345678.json": `{"version":1,"id":"read_12345678","action":"read_file","project":"runner://garage","args":{"path":"README.md"}}`,
		"relay/control-outbox/read_12345678.json": `{"version":1,"id":"read_12345678","action":"read_file","status":"completed","updated_at":"2026-08-18T10:00:00Z"}`,
		"relay/inbox/task_12345678.json": `{"version":1,"id":"task_12345678","project":"runner://family-vault","intent":"continue"}`,
		"relay/outbox/task_12345678.json": `{"version":1,"id":"task_12345678","status":"running","updated_at":"2026-08-18T10:01:00Z"}`,
	})
	items, err := parseRunnerChatActivity(raw, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("activity=%#v", items)
	}
	if items[0].ProjectRef != "runner://family-vault" || items[0].Action != "delegate_task" || items[0].State != "running" {
		t.Fatalf("autonomous activity=%#v", items[0])
	}
	if items[1].ProjectRef != "runner://garage" || items[1].Action != "read_file" || items[1].State != "completed" {
		t.Fatalf("safe-hands activity=%#v", items[1])
	}
}

func TestParseRunnerChatActivityNeverExposesHostProjectPaths(t *testing.T) {
	raw := relayActivityArchive(t, map[string]string{
		"relay/control/bad_12345678.json": `{"version":1,"id":"bad_12345678","action":"read_file","project":"/home/operator/src/private","args":{}}`,
		"relay/control-outbox/bad_12345678.json": `{"version":1,"id":"bad_12345678","action":"read_file","status":"completed","updated_at":"2026-08-18T10:00:00Z"}`,
	})
	items, err := parseRunnerChatActivity(raw, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("host path leaked into activity: %#v", items)
	}
}
