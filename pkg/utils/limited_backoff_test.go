package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/stretchr/testify/assert"
)

func TestLimitedBackOff(t *testing.T) {
	t.Run("Subject: Using limited backoff", func(t *testing.T) {
		const (
			interval   = time.Second
			retryLimit = 3
		)

		t.Run("Given a new limited backoff", func(t *testing.T) {
			backOff := NewLimitedBackOff(interval, retryLimit)

			testLimitedBackOff(t, backOff, retryLimit, interval)
		})

		t.Run("Given an exhausted limited backoff", func(t *testing.T) {
			backOff := NewLimitedBackOff(interval, retryLimit)
			callMultipleNextBackOffs(backOff, retryLimit)

			t.Run("When calling Reset", func(t *testing.T) {
				backOff.Reset()

				assertRetryCount(t, backOff, 0)

				testLimitedBackOff(t, backOff, retryLimit, interval)
			})
		})
	})
}

func testLimitedBackOff(t *testing.T, backOff *limitedBackOff, retryLimit int, interval time.Duration) {
	t.Run("When calling NextBackOff until retry limit is reached", func(t *testing.T) {
		results := callMultipleNextBackOffs(backOff, retryLimit)
		assertResultsEqualExpected(t, interval, results)
		assertRetryCount(t, backOff, retryLimit)

		const retryBeyondLimit = 10
		t.Run(fmt.Sprint("When calling NextBackOff ", retryBeyondLimit, " times beyond limit"), func(t *testing.T) {
			results := callMultipleNextBackOffs(backOff, retryBeyondLimit)
			assertResultsEqualExpected(t, backoff.Stop, results)
			assertRetryCount(t, backOff, retryLimit)
		})
	})
}

func callMultipleNextBackOffs(limitedBackOff *limitedBackOff, count int) []time.Duration {
	results := make([]time.Duration, count)
	for i := 0; i < count; i++ {
		results[i] = limitedBackOff.NextBackOff()
	}
	return results
}

func assertResultsEqualExpected(t *testing.T, expected time.Duration, results []time.Duration) {
	for _, result := range results {
		assert.Equal(t, expected, result)
	}
}

func assertRetryCount(t *testing.T, backOff *limitedBackOff, expected int) {
	assert.Equal(t, expected, backOff.RetryCount())
}
