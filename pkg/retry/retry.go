package retry

import (
	"context"
	"time"
)

type SleepFunc func(context.Context, time.Duration) error

func Sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
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

type Operation func(attempt int) (retryable bool, retryAfter time.Duration, err error)

func Do(ctx context.Context, attempts int, fallbackDelay time.Duration, sleep SleepFunc, operation Operation) error {
	if attempts < 1 {
		attempts = 1
	}
	if sleep == nil {
		sleep = Sleep
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		retryable, retryAfter, err := operation(attempt)
		if err == nil {
			return nil
		}
		lastErr = err

		if !retryable || attempt == attempts {
			return err
		}
		if retryAfter <= 0 {
			retryAfter = fallbackDelay
		}
		if err := sleep(ctx, retryAfter); err != nil {
			return err
		}
	}

	return lastErr
}
