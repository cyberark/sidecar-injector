package utils

import (
	"time"

	"github.com/cenkalti/backoff"
)

// limitedBackOff is a backoff policy that always returns the same backoff delay, for a limited number of times.
// This is an expansion of the constant backoff policy, which is unlimited.
// It implements the BackOff interface from backoff package.
type limitedBackOff struct {
	Interval   time.Duration
	RetryLimit int
	retryCount int
}

func (b *limitedBackOff) RetryCount() int {
	return b.retryCount
}

func NewLimitedBackOff(duration time.Duration, retryLimit int) *limitedBackOff {
	return &limitedBackOff{
		Interval:   duration,
		RetryLimit: retryLimit,
		retryCount: 0,
	}
}

func (b *limitedBackOff) Reset() {
	b.retryCount = 0
}

func (b *limitedBackOff) NextBackOff() time.Duration {
	if b.retryCount >= b.RetryLimit {
		return backoff.Stop
	}

	b.retryCount++
	return b.Interval
}
