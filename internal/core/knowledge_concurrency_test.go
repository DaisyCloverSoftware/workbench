package core

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentKnowledgeSavesDoNotLoseItems(t *testing.T) {
	isolateKnowledgeConfig(t)
	const count = 32
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := SaveKnowledge(KnowledgeItem{
				Scope:   ScopeGlobal,
				Kind:    KindFact,
				Title:   fmt.Sprintf("concurrent item %02d", i),
				Content: fmt.Sprintf("durable concurrent content %02d", i),
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	items, err := SearchKnowledge("", "concurrent", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != count {
		t.Fatalf("concurrent writes lost knowledge: got %d want %d", len(items), count)
	}
}

func TestConcurrentKnowledgeUsageMarksDoNotLoseCounts(t *testing.T) {
	isolateKnowledgeConfig(t)
	item, err := SaveKnowledge(KnowledgeItem{Scope: ScopeGlobal, Kind: KindRoutine, Title: "shared usage routine", Content: "reuse this routine"})
	if err != nil {
		t.Fatal(err)
	}
	const count = 40
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := MarkKnowledgeUsed(item.ID); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	items, err := SearchKnowledge("", "shared usage", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].UseCount != count {
		t.Fatalf("usage marks lost updates: %#v", items)
	}
}
