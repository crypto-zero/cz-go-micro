package client

import (
	"context"
	"time"

	"c-z.dev/go-micro/util/backoff"
)

type BackoffFunc func(ctx context.Context, req Request, attempts int) (time.Duration, error)

func exponentialBackoff(ctx context.Context, req Request, attempts int) (time.Duration, error) {
	return backoff.Do(attempts), nil
}
