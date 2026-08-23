package core

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessProgressContractCarriesCurrentAndTotal(t *testing.T) {
	body, err := os.ReadFile("harness_protocol.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `Current    int64`) || !strings.Contains(text, `Total      int64`) {
		t.Fatal("measured progress contract missing current/total")
	}
}
