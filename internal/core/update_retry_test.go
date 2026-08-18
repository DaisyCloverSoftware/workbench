package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestFetchUpdateReleaseWithRetryRecoversFromTemporaryDNSFailure(t *testing.T) {
	attempts := 0
	fetch := func(context.Context) (UpdateRelease, error) {
		attempts++
		if attempts < 3 {
			return UpdateRelease{}, fmt.Errorf("wrapped transport failure: %w", &net.DNSError{Err: "temporary failure", Name: "api.github.com", IsTemporary: true})
		}
		return UpdateRelease{Version: "0.9.12", Tag: "v0.9.12", Assets: map[string]UpdateAsset{}}, nil
	}

	release, err := fetchUpdateReleaseWithRetry(context.Background(), fetch, []time.Duration{0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || release.Version != "0.9.12" {
		t.Fatalf("attempts=%d release=%#v", attempts, release)
	}
}

func TestFetchUpdateReleaseWithRetryReturnsAvailabilityWarningAfterNetworkRetries(t *testing.T) {
	attempts := 0
	fetch := func(context.Context) (UpdateRelease, error) {
		attempts++
		return UpdateRelease{}, fmt.Errorf("wrapped transport failure: %w", &net.DNSError{Err: "temporary failure", Name: "api.github.com", IsTemporary: true})
	}

	_, err := fetchUpdateReleaseWithRetry(context.Background(), fetch, []time.Duration{0, 0})
	if err == nil || !IsUpdateCheckUnavailable(err) {
		t.Fatalf("expected update-check availability error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts=%d want 3", attempts)
	}
}

func TestFetchUpdateReleaseWithRetryDoesNotRetryTrustOrMetadataFailures(t *testing.T) {
	attempts := 0
	want := errors.New("release metadata is untrusted")
	fetch := func(context.Context) (UpdateRelease, error) {
		attempts++
		return UpdateRelease{}, want
	}

	_, err := fetchUpdateReleaseWithRetry(context.Background(), fetch, []time.Duration{0, 0})
	if !errors.Is(err, want) {
		t.Fatalf("error=%v want %v", err, want)
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d want 1", attempts)
	}
}

func TestFetchUpdateReleaseWithRetryHonorsCancellationDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	fetch := func(context.Context) (UpdateRelease, error) {
		attempts++
		cancel()
		return UpdateRelease{}, fmt.Errorf("wrapped transport failure: %w", &net.DNSError{Err: "temporary failure", Name: "api.github.com", IsTemporary: true})
	}

	_, err := fetchUpdateReleaseWithRetry(ctx, fetch, []time.Duration{time.Hour})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v want context cancellation", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d want 1", attempts)
	}
}
