package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPendingPrivateControlPathsUsesOneCompletedIndex(t *testing.T) {
	controls := make([]string, 0, 1002)
	outboxes := make([]string, 0, 1001)

	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("control-%08d", i)
		controls = append(controls, "relay/control/"+id+".json")
		if i < 998 {
			outboxes = append(outboxes, "relay/control-outbox/"+id+".json")
		}
	}

	// Malformed filenames must never enter either side of the ID index.
	controls = append(controls, "relay/control/not valid.json")
	outboxes = append(outboxes, "relay/control-outbox/not valid.json")

	// An unrelated but valid outbox entry must not remove any pending control.
	outboxes = append(outboxes, "relay/control-outbox/other-00000001.json")

	got := pendingPrivateControlPaths(controls, outboxes)
	want := []string{
		"relay/control/control-00000998.json",
		"relay/control/control-00000999.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending=%#v want %#v", got, want)
	}
}

func TestPendingPrivateControlPathsPreservesInputOrder(t *testing.T) {
	controls := []string{
		"relay/control/control-bbbbbbbb.json",
		"relay/control/control-aaaaaaaa.json",
		"relay/control/control-cccccccc.json",
	}
	outboxes := []string{
		"relay/control-outbox/control-aaaaaaaa.json",
	}

	got := pendingPrivateControlPaths(controls, outboxes)
	want := []string{
		"relay/control/control-bbbbbbbb.json",
		"relay/control/control-cccccccc.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending=%#v want %#v", got, want)
	}
}
