package core

import (
	"testing"
	"time"
)

func isolateChatActivity(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("APPDATA", root)
	t.Setenv("HOME", root)
}

func TestChatActivityRoundTripAndUpdate(t *testing.T) {
	isolateChatActivity(t)
	if err := RecordChatActivity("chatreq_12345678", "runner://family-vault", "read_file", "running"); err != nil {
		t.Fatal(err)
	}
	if err := RecordChatActivity("chatreq_12345678", "runner://family-vault", "read_file", "completed"); err != nil {
		t.Fatal(err)
	}
	items, err := RecentChatActivity(10, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].State != "completed" || items[0].ProjectRef != "runner://family-vault" {
		t.Fatalf("unexpected activity: %#v", items)
	}
}

func TestChatActivityRejectsNonRunnerProjectAndInvalidState(t *testing.T) {
	isolateChatActivity(t)
	if err := RecordChatActivity("chatreq_12345678", "/home/user/project", "read_file", "completed"); err == nil {
		t.Fatal("host path must not enter chat activity feed")
	}
	if err := RecordChatActivity("chatreq_12345678", "runner://family-vault", "read_file", "mystery"); err == nil {
		t.Fatal("invalid activity state must be rejected")
	}
}
