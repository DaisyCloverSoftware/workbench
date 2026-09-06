package core

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostBridgeTargetLoopFollowsSavedTarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	reads := 0
	var got []string
	read := func() (string, error) {
		reads++
		switch reads {
		case 1:
			return "", nil
		case 2:
			return "invalid", errors.New("invalid target")
		case 3:
			return "user@runner-a", nil
		case 4:
			return "", nil
		default:
			return "user@runner-b", nil
		}
	}
	err := runHostBridgeTargetLoop(ctx, read, func(_ context.Context, host string) error {
		got = append(got, host)
		if len(got) == 2 {
			cancel()
		}
		return nil
	}, time.Millisecond)
	if err != nil || !reflect.DeepEqual(got, []string{"user@runner-a", "user@runner-b"}) || reads != 5 {
		t.Fatalf("err=%v targets=%v reads=%d", err, got, reads)
	}
}

func TestHostBridgeTargetLoopPinsInFlightCycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var target atomic.Value
	target.Store("user@runner-a")
	started, release, done := make(chan string, 2), make(chan struct{}), make(chan error, 1)
	var active, peak atomic.Int32
	var completed []string
	go func() {
		done <- runHostBridgeTargetLoop(ctx, func() (string, error) { return target.Load().(string), nil },
			func(c context.Context, host string) error {
				n := active.Add(1)
				defer active.Add(-1)
				if n > peak.Load() {
					peak.Store(n)
				}
				started <- host
				if host == "user@runner-a" {
					select {
					case <-release:
					case <-c.Done():
						return errors.New("cycle unexpectedly cancelled")
					}
				}
				completed = append(completed, host)
				if host == "user@runner-b" {
					cancel()
				}
				return nil
			}, time.Millisecond)
	}()
	select {
	case host := <-started:
		if host != "user@runner-a" {
			t.Fatal(host)
		}
	case <-ctx.Done():
		t.Fatal("first cycle never started")
	}
	target.Store("user@runner-b")
	select {
	case host := <-started:
		t.Fatalf("new target started before old cycle completed: %s", host)
	case <-time.After(15 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("loop failed to terminate")
	}
	if peak.Load() != 1 || !reflect.DeepEqual(completed, []string{"user@runner-a", "user@runner-b"}) {
		t.Fatalf("concurrent cycles=%d completed=%v", peak.Load(), completed)
	}
}

func TestHostBridgeTargetLoopCancelledBeforeRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runHostBridgeTargetLoop(ctx, func() (string, error) { t.Fatal("read after cancellation"); return "", nil },
		func(context.Context, string) error { t.Fatal("poll after cancellation"); return nil }, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHostBridgeTargetLoopCancelledDuringRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := runHostBridgeTargetLoop(ctx, func() (string, error) { cancel(); return "user@runner", nil },
		func(context.Context, string) error { t.Fatal("poll after cancellation"); return nil }, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHostBridgeTargetLoopCancellationInterruptsWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	polled, done := make(chan struct{}), make(chan error, 1)
	go func() {
		done <- runHostBridgeTargetLoop(ctx, func() (string, error) { return "user@runner", nil },
			func(context.Context, string) error { close(polled); return nil }, time.Hour)
	}()
	select {
	case <-polled:
	case <-time.After(2 * time.Second):
		t.Fatal("poll did not run")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait ignored cancellation")
	}
}

func TestHostBridgeTargetLoopPropagatesLocalFailure(t *testing.T) {
	want := errors.New("local identity initialization failed")
	err := runHostBridgeTargetLoop(context.Background(), func() (string, error) { return "user@runner", nil },
		func(context.Context, string) error { return want }, time.Hour)
	if !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
}

func TestHostBridgeTargetLoopRejectsInvalidWiring(t *testing.T) {
	read := func() (string, error) { return "user@runner", nil }
	poll := func(context.Context, string) error { return nil }
	for _, tc := range []struct {
		name  string
		read  func() (string, error)
		poll  func(context.Context, string) error
		delay time.Duration
	}{
		{"nil-source", nil, poll, time.Second}, {"nil-poll", read, nil, time.Second},
		{"zero-delay", read, poll, 0}, {"negative-delay", read, poll, -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if runHostBridgeTargetLoop(context.Background(), tc.read, tc.poll, tc.delay) == nil {
				t.Fatal("invalid loop configuration accepted")
			}
		})
	}
}
