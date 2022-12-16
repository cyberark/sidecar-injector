package secrets

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/clients/conjur"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/config"
	k8sSecretsStorage "github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/k8s_secrets_storage"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/pushtofile"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/utils"
)

const (
	secretProviderGracePeriod = time.Duration(10 * time.Millisecond)
)

// CommonProviderConfig provides config that is common to all providers
type CommonProviderConfig struct {
	StoreType       string
	SanitizeEnabled bool
}

// ProviderConfig provides the configuration necessary to create a secrets
// Provider.
type ProviderConfig struct {
	CommonProviderConfig
	k8sSecretsStorage.K8sProviderConfig
	pushtofile.P2FProviderConfig
}

// ProviderFunc describes a function type responsible for providing secrets to
// an unspecified target. It returns either an error, or a flag that indicates
// whether any target secret files or Kubernetes Secrets have been updated.
type ProviderFunc func() (updated bool, err error)

// RepeatableProviderFunc describes a function type that is capable of looping
// indefinitely while providing secrets to unspecified targets.
type RepeatableProviderFunc func() error

// NewProviderForType returns a ProviderFunc responsible for providing secrets in a given mode.
func NewProviderForType(
	traceContext context.Context,
	secretsRetrieverFunc conjur.RetrieveSecretsFunc,
	providerConfig ProviderConfig,
) (ProviderFunc, []error) {
	switch providerConfig.StoreType {
	case config.K8s:
		provider := k8sSecretsStorage.NewProvider(
			traceContext,
			secretsRetrieverFunc,
			providerConfig.CommonProviderConfig.SanitizeEnabled,
			providerConfig.K8sProviderConfig,
		)
		return provider.Provide, nil
	case config.File:
		provider, err := pushtofile.NewProvider(
			secretsRetrieverFunc,
			providerConfig.CommonProviderConfig.SanitizeEnabled,
			providerConfig.P2FProviderConfig,
		)
		if err != nil {
			return nil, err
		}
		provider.SetTraceContext(traceContext)
		return provider.Provide, nil
	default:
		return nil, []error{fmt.Errorf(
			messages.CSPFK054E,
			providerConfig.StoreType,
		)}
	}
}

// RetryableSecretProvider returns a new ProviderFunc, which wraps the provided ProviderFunc
// in a limitedBackOff-restricted Retry call.
func RetryableSecretProvider(
	retryInterval time.Duration,
	retryCountLimit int,
	provideSecrets ProviderFunc,
) ProviderFunc {
	limitedBackOff := utils.NewLimitedBackOff(
		retryInterval,
		retryCountLimit,
	)

	return func() (bool, error) {
		var updated bool
		var retErr error

		err := backoff.Retry(func() error {
			if limitedBackOff.RetryCount() > 0 {
				log.Info(fmt.Sprintf(messages.CSPFK010I, limitedBackOff.RetryCount(), limitedBackOff.RetryLimit))
			}
			updated, retErr = provideSecrets()
			return retErr
		}, limitedBackOff)

		if err != nil {
			log.Error(messages.CSPFK038E, err)
		}
		return updated, err
	}
}

// ProviderRefreshConfig specifies the secret refresh configuration
// for a repeatable secret provider.
type ProviderRefreshConfig struct {
	Mode                  string
	SecretRefreshInterval time.Duration
	ProviderQuit          chan struct{}
}

// RepeatableSecretProvider returns a new ProviderFunc, which wraps a retryable
// ProviderFunc inside a function that operates in one of three modes:
//  - Run once and return (for init or application container modes)
//  - Run once and sleep forever (for sidecar mode without periodic refresh)
//  - Run periodically (for sidecar mode with periodic refresh)
func RepeatableSecretProvider(
	refreshConfig ProviderRefreshConfig,
	provideSecrets ProviderFunc,
) RepeatableProviderFunc {
	return repeatableSecretProvider(refreshConfig, provideSecrets,
		defaultStatusUpdater)
}

func repeatableSecretProvider(
	config ProviderRefreshConfig,
	provideSecrets ProviderFunc,
	status statusUpdater,
) RepeatableProviderFunc {

	var periodicQuit = make(chan struct{})
	var periodicError = make(chan error)
	var ticker *time.Ticker
	var err error

	return func() error {
		if err = status.copyScripts(); err != nil {
			return err
		}
		if _, err = provideSecrets(); err != nil {
			// Return immediately upon error, regardless of operating mode
			return err
		}
		err = status.setSecretsProvided()
		if err != nil {
			return err
		}
		switch {
		case config.Mode != "sidecar":
			// Run once and return if not in sidecar mode
			return nil
		case config.SecretRefreshInterval > 0:
			// Run periodically if in sidecar mode with periodic refresh
			ticker = time.NewTicker(config.SecretRefreshInterval)
			config := periodicConfig{
				ticker:        ticker,
				periodicQuit:  periodicQuit,
				periodicError: periodicError,
			}
			go periodicSecretProvider(provideSecrets, config, status)
		default:
			// Run once and sleep forever if in sidecar mode without
			// periodic refresh (fall through)
		}

		// Wait here for a signal to quit providing secrets or an error
		// from the periodicSecretProvider() function
		select {
		case <-config.ProviderQuit:
			break
		case err = <-periodicError:
			break
		}

		// Allow the periodicSecretProvider goroutine to gracefully shut down
		if config.SecretRefreshInterval > 0 {
			// Kill the ticker
			ticker.Stop()
			periodicQuit <- struct{}{}
			// Let the go routine exit
			time.Sleep(secretProviderGracePeriod)
		}
		return err
	}
}

type periodicConfig struct {
	ticker        *time.Ticker
	periodicQuit  <-chan struct{}
	periodicError chan<- error
}

func periodicSecretProvider(
	provideSecrets ProviderFunc,
	config periodicConfig,
	status statusUpdater,
) {
	for {
		select {
		case <-config.periodicQuit:
			return
		case <-config.ticker.C:
			updated, err := provideSecrets()
			if err == nil && updated {
				err = status.setSecretsUpdated()
			}
			if err != nil {
				config.periodicError <- err
			}
		}
	}
}
