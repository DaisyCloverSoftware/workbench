package core

import (
	"context"
	"errors"
	"net"
	"time"
)

var officialUpdateRetryDelays = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond}

// UpdateCheckUnavailableError means Workbench could not reach the official
// update service after bounded retries. The current installation is untouched;
// callers should present this as a temporary availability warning, not as a
// failed installation.
type UpdateCheckUnavailableError struct {
	Err error
}

func (e *UpdateCheckUnavailableError) Error() string {
	if e == nil || e.Err == nil {
		return "Workbench update check is temporarily unavailable"
	}
	return "Workbench update check is temporarily unavailable: " + e.Err.Error()
}

func (e *UpdateCheckUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func IsUpdateCheckUnavailable(err error) bool {
	var target *UpdateCheckUnavailableError
	return errors.As(err, &target)
}

// FetchOfficialLatestReleaseResilient preserves all existing official-release
// trust checks while retrying only transport-level failures such as temporary
// DNS resolution or connection errors. Metadata/trust failures remain hard
// errors and are never retried away.
func FetchOfficialLatestReleaseResilient(ctx context.Context) (UpdateRelease, error) {
	return fetchUpdateReleaseWithRetry(ctx, FetchOfficialLatestRelease, officialUpdateRetryDelays)
}

func fetchUpdateReleaseWithRetry(ctx context.Context, fetch func(context.Context) (UpdateRelease, error), delays []time.Duration) (UpdateRelease, error) {
	if fetch == nil {
		return UpdateRelease{}, errors.New("update fetch function is nil")
	}
	var lastErr error
	for attempt := 0; ; attempt++ {
		release, err := fetch(ctx)
		if err == nil {
			return release, nil
		}
		lastErr = err
		if !isTransientUpdateTransportError(err) {
			return UpdateRelease{}, err
		}
		if attempt >= len(delays) {
			return UpdateRelease{}, &UpdateCheckUnavailableError{Err: lastErr}
		}
		if err := waitForUpdateRetry(ctx, delays[attempt]); err != nil {
			return UpdateRelease{}, err
		}
	}
}

func isTransientUpdateTransportError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var networkErr net.Error
	return errors.As(err, &networkErr)
}

func waitForUpdateRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
