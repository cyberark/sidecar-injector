package secrets

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	logger "github.com/cyberark/conjur-authn-k8s-client/pkg/log"
	"github.com/stretchr/testify/assert"
)

const (
	providerDelayMsecs      = 5
	largeProviderDelayMsecs = 50
	atomicWriteDelayMsecs   = 3
)

// call count state, so that test cases don't interfere with one another.
type mockProvider struct {
	calledCount          int
	callLatencyMsecs     time.Duration
	injectFailure        bool
	failOnCountN         int
	injectInitialFailure bool
	failUntilCountN      int
	targetsUpdated       bool
}

func (m *mockProvider) provide() (bool, error) {
	m.calledCount++
	switch {
	case m.injectFailure && (m.calledCount >= m.failOnCountN):
		return m.targetsUpdated, errors.New("Failed to Provide")
	case m.injectInitialFailure && (m.calledCount < m.failUntilCountN):
		return m.targetsUpdated, errors.New("Failed to Provide")
	case m.callLatencyMsecs > 0:
		time.Sleep(m.callLatencyMsecs * time.Millisecond)
	}
	return m.targetsUpdated, nil
}

func (m *mockProvider) count() int {
	return m.calledCount
}

func goodProvider() *mockProvider {
	return &mockProvider{}
}

func goodProviderTargetsUpdated() *mockProvider {
	return &mockProvider{targetsUpdated: true}
}

func badProvider() *mockProvider {
	return &mockProvider{injectFailure: true, failOnCountN: 1}
}

func eventualProvider(failUntilCountN int) *mockProvider {
	return &mockProvider{injectInitialFailure: true, failUntilCountN: failUntilCountN}
}

func eventualProviderTargetsUpdated(failUntilCountN int) *mockProvider {
	return &mockProvider{
		injectInitialFailure: true,
		failUntilCountN:      failUntilCountN,
		targetsUpdated:       true,
	}
}

func slowProvider(latencyMsecs time.Duration) *mockProvider {
	return &mockProvider{callLatencyMsecs: latencyMsecs}
}
func goodAtFirstThenBadProvider(failOnCountN int) *mockProvider {
	return &mockProvider{injectFailure: true, failOnCountN: failOnCountN}
}

func TestRetryableSecretProvider(t *testing.T) {
	TestCases := []struct {
		description   string
		interval      time.Duration
		limit         int
		provider      *mockProvider
		assertOn      string
		expectUpdated bool
	}{
		{
			description: "the ProviderFunc can return within the retry limit",
			interval:    time.Duration(10) * time.Millisecond,
			limit:       3,
			provider:    eventualProvider(3),
			assertOn:    "success",
		},
		{
			description:   "the ProviderFunc can return with target updated status",
			interval:      time.Duration(10) * time.Millisecond,
			limit:         3,
			provider:      eventualProviderTargetsUpdated(3),
			assertOn:      "success",
			expectUpdated: true,
		},
		{
			description: "the ProviderFunc is retried the proper amount of times",
			interval:    time.Duration(10) * time.Millisecond,
			limit:       2,
			provider:    badProvider(),
			assertOn:    "count",
		},
		{
			description: "the ProviderFunc is only retried after the proper duration",
			interval:    time.Duration(20) * time.Millisecond,
			limit:       2,
			provider:    badProvider(),
			assertOn:    "interval",
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.description, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger.InfoLogger = log.New(&logBuffer, "", 0)

			retryableProvider := RetryableSecretProvider(
				tc.interval, tc.limit, tc.provider.provide,
			)

			start := time.Now()
			updated, err := retryableProvider()
			duration := time.Since(start)

			logMessages := logBuffer.String()
			switch tc.assertOn {
			case "count":
				assert.NotNil(t, err)
				assert.Contains(t, logMessages,
					fmt.Sprintf("CSPFK010I Updating Kubernetes Secrets: %d retries out of %d", tc.limit, tc.limit))
			case "interval":
				assert.NotNil(t, err)
				maxDuration := (time.Duration(tc.limit) * time.Millisecond * tc.interval) + (time.Duration(1) * time.Millisecond)
				assert.LessOrEqual(t, duration, maxDuration)
			case "success":
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectUpdated, updated)
		})
	}
}

func TestRepeatableSecretProvider(t *testing.T) {
	testCases := []struct {
		description    string
		mode           string
		interval       time.Duration
		testTime       time.Duration // total test time for all tests must be less than 3m
		expectedCount  int
		expectProvided bool
		expectUpdated  bool
		provider       *mockProvider
		assertOn       string
		targetsUpdated bool
	}{
		{
			description:    "init container, happy path",
			mode:           "init",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(25) * time.Millisecond,
			expectedCount:  1,
			expectProvided: true,
			expectUpdated:  false,
			provider:       goodProvider(),
			assertOn:       "success",
		},
		{
			description:    "sidecar container, happy path, no targets updated",
			mode:           "sidecar",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(65+atomicWriteDelayMsecs) * time.Millisecond,
			expectedCount:  7,
			expectProvided: true,
			expectUpdated:  false,
			provider:       goodProvider(),
			assertOn:       "success",
		},
		{
			description:    "sidecar container, happy path, targets updated",
			mode:           "sidecar",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(65+atomicWriteDelayMsecs) * time.Millisecond,
			expectedCount:  7,
			expectProvided: true,
			expectUpdated:  true,
			provider:       goodProviderTargetsUpdated(),
			assertOn:       "success",
			targetsUpdated: true,
		},
		{
			description:    "sidecar with zero duration",
			mode:           "sidecar",
			interval:       time.Duration(0) * time.Millisecond,
			testTime:       time.Duration(25) * time.Millisecond,
			expectedCount:  1,
			expectProvided: true,
			expectUpdated:  false,
			provider:       goodProvider(),
			assertOn:       "success",
		},
		{
			description:    "application mode, happy path",
			mode:           "application",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(25) * time.Millisecond,
			expectProvided: true,
			expectUpdated:  false,
			expectedCount:  1,
			provider:       goodProvider(),
			assertOn:       "success",
		},
		{
			// The provider is slow, but still finishes within the interval time.
			// Note the ticker is started after the first provideSecrets call so
			// that delay is added to the duration
			description:    "sidecar with slow provider",
			mode:           "sidecar",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(65+providerDelayMsecs) * time.Millisecond,
			expectedCount:  7,
			expectProvided: true,
			expectUpdated:  false,
			provider:       slowProvider(providerDelayMsecs),
			assertOn:       "success",
		},
		{
			// In this test the provider takes longer to run than the
			// interval time. The Go timer will adjust the time interval due to the
			// slower receiver. Secrets Provider is expected to be called every largeProviderDelayMsecs
			// The first timer tick is delayed by duration so that is added to the test time.
			description:    "sidecar with duration less than fetch time",
			mode:           "sidecar",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(325+10) * time.Millisecond,
			expectedCount:  7,
			expectProvided: true,
			expectUpdated:  false,
			provider:       slowProvider(largeProviderDelayMsecs),
			assertOn:       "success",
		},
		{
			description:    "badProvider for init container",
			mode:           "init",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(25) * time.Millisecond,
			expectedCount:  1,
			expectProvided: false,
			expectUpdated:  false,
			provider:       badProvider(),
			assertOn:       "fail",
		},
		{
			description:    "badProvider for sidecar",
			mode:           "sidecar",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(25) * time.Millisecond,
			expectedCount:  1,
			expectProvided: false,
			expectUpdated:  false,
			provider:       badProvider(),
			assertOn:       "fail",
		},
		{
			description:    "goodAtFirstThenBadProvider for sidecar",
			mode:           "sidecar",
			interval:       time.Duration(10) * time.Millisecond,
			testTime:       time.Duration(35) * time.Millisecond,
			expectedCount:  1,
			expectProvided: true,
			expectUpdated:  false,
			provider:       goodAtFirstThenBadProvider(2),
			assertOn:       "fail",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var err error

			// Create a test provider status updater
			updater, err := newTestStatusUpdater(injectErrs{})
			assert.NoError(t, err)
			defer updater.cleanup()
			fileUpdater := updater.fileUpdater

			// Construct a secret provider function
			var providerQuit = make(chan struct{})
			refreshConfig := ProviderRefreshConfig{
				Mode:                  tc.mode,
				SecretRefreshInterval: tc.interval,
				ProviderQuit:          providerQuit,
			}
			provideSecrets := repeatableSecretProvider(
				refreshConfig, tc.provider.provide, fileUpdater)

			// Run the secrets provider
			testError := make(chan error)
			go func() {
				err := provideSecrets()
				testError <- err
			}()
			select {
			case err = <-testError:
				break
			case <-time.After(tc.testTime):
				providerQuit <- struct{}{}
				break
			}

			// Check results
			if err == nil {
				assert.Equal(t, tc.expectedCount, tc.provider.count())
			}

			assert.FileExists(t, updater.targetScriptFile)
			switch tc.expectProvided {
			case true:
				assert.FileExists(t, fileUpdater.providedFile)
			case false:
				assert.NoFileExists(t, fileUpdater.providedFile)
			}
			switch tc.expectUpdated {
			case true:
				assert.FileExists(t, fileUpdater.updatedFile)
			case false:
				assert.NoFileExists(t, fileUpdater.updatedFile)
			}

			switch tc.assertOn {
			case "success":
				assert.NoError(t, err)
			case "fail":
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), "Failed to Provide")
			}
		})
	}
}
