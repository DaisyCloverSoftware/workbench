package core

import "testing"

func TestChangesetPathParsingPreservesWhitespace(t *testing.T) {
	got := splitNULPaths(" spaced name .txt\x00normal.txt\x00")
	if len(got) != 2 || got[0] != " spaced name .txt" || got[1] != "normal.txt" {
		t.Fatalf("unexpected paths: %#v", got)
	}
}

func TestLimitedCaptureKeepsDrainingPastLimit(t *testing.T) {
	w := &limitedCapture{limit: 4}
	n, err := w.Write([]byte("abcdefgh"))
	if err != nil || n != 8 {
		t.Fatalf("write=%d err=%v", n, err)
	}
	if !w.exceeded || w.String() != "abcd" {
		t.Fatalf("unexpected capture: exceeded=%t text=%q", w.exceeded, w.String())
	}
}
