package core

import (
	"fmt"
	"sync"
	"testing"
)

func TestRelayStateConcurrentSavesDoNotLoseRecords(t *testing.T) {
	isolateKnowledgeConfig(t)
	const count = 40
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- SaveRelayRecord(RelayRecord{
				RelayID:         fmt.Sprintf("relay-%02d", i),
				Source:          "test",
				WorkbenchTaskID: fmt.Sprintf("task-%02d", i),
				Project:         "sample",
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	records, err := ListRelayRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != count {
		t.Fatalf("saved %d records, got %d", count, len(records))
	}
}

func TestRelayStateUpdatePreservesCreatedAtAndAnswerDigest(t *testing.T) {
	isolateKnowledgeConfig(t)
	if err := SaveRelayRecord(RelayRecord{RelayID: "relay-one", Source: "test", LastAnswerDigest: "digest-one"}); err != nil {
		t.Fatal(err)
	}
	first, ok, err := LoadRelayRecord("relay-one")
	if err != nil || !ok {
		t.Fatalf("load first: ok=%v err=%v", ok, err)
	}
	if err := SaveRelayRecord(RelayRecord{RelayID: "relay-one", Source: "test", WorkbenchTaskID: "task-one"}); err != nil {
		t.Fatal(err)
	}
	second, ok, err := LoadRelayRecord("relay-one")
	if err != nil || !ok {
		t.Fatalf("load second: ok=%v err=%v", ok, err)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("created_at changed: %v -> %v", first.CreatedAt, second.CreatedAt)
	}
	if second.LastAnswerDigest != "digest-one" {
		t.Fatalf("answer digest was lost: %#v", second)
	}
}
