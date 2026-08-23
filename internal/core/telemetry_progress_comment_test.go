package core

import (
	"os"
	"strings"
	"testing"
)

func TestTaskProgressCommentExplainsNoGuessedPercentage(t *testing.T) {
	body, err := os.ReadFile("work_item.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), "rather than a guessed percentage") {
		t.Fatal("no-guessed-percentage contract comment missing")
	}
}
