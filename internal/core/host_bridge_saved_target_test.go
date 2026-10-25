package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostBridgeSavedTargetInitialValue(t *testing.T) {
	var empty SavedHostBridgeTarget
	if empty.Current() != "" {
		t.Fatal("zero value must be dormant")
	}
	if got := NewSavedHostBridgeTarget("  user@runner-a \n").Current(); got != "user@runner-a" {
		t.Fatal(got)
	}
}

func TestHostBridgeSavedTargetRejectsFailedPersistence(t *testing.T) {
	target := NewSavedHostBridgeTarget("user@runner-a")
	want := errors.New("save failed")
	memory := "user@runner-a"
	calls := 0
	err := target.Save("user@runner-b", func() error {
		calls++
		// Engine.SavePreferences can change memory before persistence fails.
		memory = "user@runner-b"
		if target.Current() != "user@runner-a" {
			t.Fatal("unpersisted target escaped")
		}
		return want
	})
	if !errors.Is(err, want) || calls != 1 || memory != "user@runner-b" || target.Current() != "user@runner-a" {
		t.Fatalf("failed save changed connection authority: %v %d %s", err, calls, target.Current())
	}
}

func TestHostBridgeSavedTargetPublishesAfterSuccessfulSave(t *testing.T) {
	target := NewSavedHostBridgeTarget("")
	err := target.Save(" user@runner-a ", func() error {
		if target.Current() != "" {
			t.Fatal("published before persistence")
		}
		return nil
	})
	if err != nil || target.Current() != "user@runner-a" {
		t.Fatalf("%v %s", err, target.Current())
	}
}

func TestHostBridgeSavedTargetFailedClearPreservesConnection(t *testing.T) {
	target := NewSavedHostBridgeTarget("user@runner-a")
	if err := target.Save("", func() error { return errors.New("disk full") }); err == nil {
		t.Fatal("expected failure")
	}
	if target.Current() != "user@runner-a" {
		t.Fatal("failed clear disconnected bridge")
	}
	if err := target.Save(" \t", func() error { return nil }); err != nil || target.Current() != "" {
		t.Fatal("persisted clear must suspend new polls")
	}
}

func TestHostBridgeSavedTargetNilPersistence(t *testing.T) {
	target := NewSavedHostBridgeTarget("user@runner-a")
	if err := target.Save("user@runner-b", nil); err == nil || target.Current() != "user@runner-a" {
		t.Fatal("nil callback changed target")
	}
}

func TestHostBridgeSavedTargetReadsRemainAvailableDuringSave(t *testing.T) {
	target := NewSavedHostBridgeTarget("user@runner-a")
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() { done <- target.Save("user@runner-b", func() error { close(entered); <-release; return nil }) }()
	defer func() { close(release); <-done }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("save did not start")
	}
	read := make(chan string, 1)
	go func() { read <- target.Current() }()
	select {
	case got := <-read:
		if got != "user@runner-a" {
			t.Fatal("unpersisted value visible")
		}
	case <-time.After(time.Second):
		t.Fatal("persistence blocked bridge reader")
	}
}

func TestHostBridgeSavedTargetSerializesConcurrentSaves(t *testing.T) {
	target := NewSavedHostBridgeTarget("user@runner-a")
	var active, overlaps atomic.Int32
	var persisted string
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			host := "user@runner-b"
			if i%2 == 0 {
				host = "user@runner-c"
			}
			err := target.Save(host, func() error {
				if active.Add(1) != 1 {
					overlaps.Add(1)
				}
				persisted = host
				active.Add(-1)
				return nil
			})
			if err != nil {
				t.Error(err)
			}
			_ = target.Current()
		}(i)
	}
	wg.Wait()
	if overlaps.Load() != 0 || target.Current() != persisted {
		t.Fatal("save completion order lost")
	}
}
