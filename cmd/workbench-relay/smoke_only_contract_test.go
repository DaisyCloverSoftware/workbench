package main

import (
	"os"
	"strings"
	"testing"
)

func TestSmokeOnlyReturnsBeforeRelayPollLoop(t *testing.T) {
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	flagIndex := strings.Index(text, `flag.Bool("smoke-only"`)
	returnIndex := strings.Index(text, "if *smokeOnly {")
	pollIndex := strings.Index(text, "for {\n\t\tif err := poll(")
	if flagIndex < 0 || returnIndex < 0 || pollIndex < 0 {
		t.Fatalf("relay smoke/poll contract markers missing: flag=%d return=%d poll=%d", flagIndex, returnIndex, pollIndex)
	}
	if !(flagIndex < returnIndex && returnIndex < pollIndex) {
		t.Fatalf("smoke-only must return before the poll loop: flag=%d return=%d poll=%d", flagIndex, returnIndex, pollIndex)
	}
	if !strings.Contains(text[returnIndex:pollIndex], "relay queue was not polled") {
		t.Fatal("smoke-only path should make its non-consuming behavior explicit")
	}
}
