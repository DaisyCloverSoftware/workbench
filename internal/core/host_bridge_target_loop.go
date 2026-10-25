package core

import (
	"context"
	"errors"
	"time"
)

// runHostBridgeTargetLoop samples the saved, validated SSH target between
// complete poll/execute/report cycles. It never restarts a running cycle merely
// because preferences changed. A blank or invalid setting disables new polls
// without terminating the desktop-owned loop. readTarget must return an error
// for invalid settings; poll must retain this target through job completion.
func runHostBridgeTargetLoop(ctx context.Context, readTarget func() (string, error), poll func(context.Context, string) error, interval time.Duration) error {
	if readTarget == nil || poll == nil || interval <= 0 {
		return errors.New("host bridge loop requires target source, poll and positive interval")
	}
	for {
		if ctx.Err() != nil {
			return nil
		}
		target, targetErr := readTarget()
		if ctx.Err() != nil {
			return nil
		}
		if targetErr == nil && target != "" {
			if err := poll(ctx, target); err != nil {
				return err
			}
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
