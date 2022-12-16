package k8ssecretsstorage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"
	"go.opentelemetry.io/otel"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"

	"github.com/cyberark/conjur-opentelemetry-tracer/pkg/trace"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/clients/conjur"
	k8sClient "github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/clients/k8s"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/config"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/utils"
)

// Secrets that have been retrieved from Conjur may need to be updated in
// more than one Kubernetes Secrets, and each Kubernetes Secret may refer to
// the application secret with a different name. The updateDestination struct
// represents one destination to which a retrieved Conjur secret value needs
// to be written when Kubernetes Secrets are updated.
type updateDestination struct {
	k8sSecretName string
	secretName    string
}

type k8sSecretsState struct {
	// Maps a K8s Secret name to the original K8s Secret API object fetched
	// from K8s.
	originalK8sSecrets map[string]*v1.Secret

	// Maps a Conjur variable ID (policy path) to all the updateDestination
	// targets which will need to be updated with the corresponding Conjur
	// secret value after it has been retrieved.
	updateDestinations map[string][]updateDestination
}

type k8sAccessDeps struct {
	retrieveSecret k8sClient.RetrieveK8sSecretFunc
	updateSecret   k8sClient.UpdateK8sSecretFunc
}

type conjurAccessDeps struct {
	retrieveSecrets conjur.RetrieveSecretsFunc
}

type logFunc func(message string, args ...interface{})
type logFuncWithErr func(message string, args ...interface{}) error
type logDeps struct {
	recordedError logFuncWithErr
	logError      logFunc
	warn          logFunc
	info          logFunc
	debug         logFunc
}

type k8sProviderDeps struct {
	k8s    k8sAccessDeps
	conjur conjurAccessDeps
	log    logDeps
}

// K8sProvider is the secret provider to be used for K8s Secrets mode. It
// makes secrets available to applications by:
// - Retrieving a list of required K8s Secrets
// - Retrieving all Conjur secrets that are referenced (via variable ID,
//   a.k.a. policy path) by those K8s Secrets.
// - Updating the K8s Secrets by replacing each Conjur variable ID
//   with the corresponding secret value that was retrieved from Conjur.
type K8sProvider struct {
	k8s                k8sAccessDeps
	conjur             conjurAccessDeps
	log                logDeps
	podNamespace       string
	requiredK8sSecrets []string
	secretsState       k8sSecretsState
	traceContext       context.Context
	sanitizeEnabled    bool
	//prevSecretsChecksums maps a k8s secret name to a sha256 checksum of the
	// corresponding secret content. This is used to detect changes in
	// secret content.
	prevSecretsChecksums map[string]utils.Checksum
}

// K8sProviderConfig provides config specific to Kubernetes Secrets provider
type K8sProviderConfig struct {
	PodNamespace       string
	RequiredK8sSecrets []string
}

// NewProvider creates a new secret provider for K8s Secrets mode.
func NewProvider(
	traceContext context.Context,
	retrieveConjurSecrets conjur.RetrieveSecretsFunc,
	sanitizeEnabled bool,
	config K8sProviderConfig,
) K8sProvider {
	return newProvider(
		k8sProviderDeps{
			k8s: k8sAccessDeps{
				k8sClient.RetrieveK8sSecret,
				k8sClient.UpdateK8sSecret,
			},
			conjur: conjurAccessDeps{
				retrieveConjurSecrets,
			},
			log: logDeps{
				log.RecordedError,
				log.Error,
				log.Warn,
				log.Info,
				log.Debug,
			},
		},
		sanitizeEnabled,
		config,
		traceContext)
}

// newProvider creates a new secret provider for K8s Secrets mode
// using dependencies provided for retrieving and updating Kubernetes
// Secrets objects.
func newProvider(
	providerDeps k8sProviderDeps,
	sanitizeEnabled bool,
	config K8sProviderConfig,
	traceContext context.Context,
) K8sProvider {
	return K8sProvider{
		k8s:                providerDeps.k8s,
		conjur:             providerDeps.conjur,
		log:                providerDeps.log,
		podNamespace:       config.PodNamespace,
		requiredK8sSecrets: config.RequiredK8sSecrets,
		sanitizeEnabled:    sanitizeEnabled,
		secretsState: k8sSecretsState{
			originalK8sSecrets: map[string]*v1.Secret{},
			updateDestinations: map[string][]updateDestination{},
		},
		traceContext:         traceContext,
		prevSecretsChecksums: map[string]utils.Checksum{},
	}
}

// Provide implements a ProviderFunc to retrieve and push secrets to K8s secrets.
func (p K8sProvider) Provide() (bool, error) {
	// Use the global TracerProvider
	tr := trace.NewOtelTracer(otel.Tracer("secrets-provider"))
	// Retrieve required K8s Secrets and parse their Data fields.
	if err := p.retrieveRequiredK8sSecrets(tr); err != nil {
		return false, p.log.recordedError(messages.CSPFK021E)
	}

	// Retrieve Conjur secrets for all K8s Secrets.
	var updated bool
	retrievedConjurSecrets, err := p.retrieveConjurSecrets(tr)
	if err != nil {
		// Delete K8s secrets for Conjur variables that no longer exist or the user no longer has permissions to.
		// In the future we'll delete only the secrets that are revoked, but for now we delete all secrets in
		// the group because we don't have a way to determine which secrets are revoked.
		if (strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "404")) && p.sanitizeEnabled {
			updated = true
			rmErr := p.removeDeletedSecrets(tr)
			if rmErr != nil {
				p.log.recordedError(messages.CSPFK063E)
				// Don't return here - continue processing
			}
		}

		return updated, p.log.recordedError(messages.CSPFK034E, err.Error())
	}

	// Update all K8s Secrets with the retrieved Conjur secrets.
	updated, err = p.updateRequiredK8sSecrets(retrievedConjurSecrets, tr)
	if err != nil {
		return updated, p.log.recordedError(messages.CSPFK023E)
	}

	p.log.info(messages.CSPFK009I)
	return updated, nil
}

func (p K8sProvider) removeDeletedSecrets(tr trace.Tracer) error {
	log.Info(messages.CSPFK021I)
	emptySecrets := make(map[string][]byte)
	variablesToDelete, err := p.listConjurSecretsToFetch()
	if err != nil {
		return err
	}
	for _, secret := range variablesToDelete {
		emptySecrets[secret] = []byte("")
	}
	_, err = p.updateRequiredK8sSecrets(emptySecrets, tr)
	if err != nil {
		return err
	}
	return nil
}

// retrieveRequiredK8sSecrets retrieves all K8s Secrets that need to be
// managed/updated by the Secrets Provider.
func (p K8sProvider) retrieveRequiredK8sSecrets(tracer trace.Tracer) error {
	spanCtx, span := tracer.Start(p.traceContext, "Gather required K8s Secrets")
	defer span.End()

	for _, k8sSecretName := range p.requiredK8sSecrets {
		_, childSpan := tracer.Start(spanCtx, "Retrieve K8s Secret")
		defer childSpan.End()
		if err := p.retrieveRequiredK8sSecret(k8sSecretName); err != nil {
			childSpan.RecordErrorAndSetStatus(err)
			span.RecordErrorAndSetStatus(err)
			return err
		}
	}
	return nil
}

// retrieveRequiredK8sSecret retrieves an individual K8s Secrets that needs
// to be managed/updated by the Secrets Provider.
func (p K8sProvider) retrieveRequiredK8sSecret(k8sSecretName string) error {

	// Retrieve the K8s Secret
	k8sSecret, err := p.k8s.retrieveSecret(p.podNamespace, k8sSecretName)
	if err != nil {
		// Error messages returned from K8s should be printed only in debug mode
		p.log.debug(messages.CSPFK004D, err.Error())
		return p.log.recordedError(messages.CSPFK020E)
	}

	// Record the K8s Secret API object
	p.secretsState.originalK8sSecrets[k8sSecretName] = k8sSecret

	// Read the value of the "conjur-map" entry in the K8s Secret's Data
	// field, if it exists. If the entry does not exist or has a null
	// value, return an error.
	conjurMapKey := config.ConjurMapKey
	conjurSecretsYAML, entryExists := k8sSecret.Data[conjurMapKey]
	if !entryExists {
		p.log.debug(messages.CSPFK008D, k8sSecretName, conjurMapKey)
		return p.log.recordedError(messages.CSPFK028E, k8sSecretName)
	}
	if len(conjurSecretsYAML) == 0 {
		p.log.debug(messages.CSPFK006D, k8sSecretName, conjurMapKey)
		return p.log.recordedError(messages.CSPFK028E, k8sSecretName)
	}

	// Parse the YAML-formatted Conjur secrets mapping that has been
	// retrieved from this K8s Secret.
	p.log.debug(messages.CSPFK009D, conjurMapKey, k8sSecretName)
	return p.parseConjurSecretsYAML(conjurSecretsYAML, k8sSecretName)
}

// Parse the YAML-formatted Conjur secrets mapping that has been retrieved
// from a K8s Secret. This secrets mapping uses application secret names
// as keys and Conjur variable IDs (a.k.a. policy paths) as values.
func (p K8sProvider) parseConjurSecretsYAML(secretsYAML []byte,
	k8sSecretName string) error {

	conjurMap := map[string]string{}
	if err := yaml.Unmarshal(secretsYAML, &conjurMap); err != nil {
		p.log.debug(messages.CSPFK007D, k8sSecretName, config.ConjurMapKey, err.Error())
		return p.log.recordedError(messages.CSPFK028E, k8sSecretName)
	}
	if len(conjurMap) == 0 {
		p.log.debug(messages.CSPFK007D, k8sSecretName, config.ConjurMapKey, "value is empty")
		return p.log.recordedError(messages.CSPFK028E, k8sSecretName)
	}

	for secretName, varID := range conjurMap {
		dest := updateDestination{k8sSecretName, secretName}
		p.secretsState.updateDestinations[varID] =
			append(p.secretsState.updateDestinations[varID], dest)
	}

	return nil
}

func (p K8sProvider) listConjurSecretsToFetch() ([]string, error) {
	updateDests := p.secretsState.updateDestinations
	if len(updateDests) == 0 {
		return nil, p.log.recordedError(messages.CSPFK025E)
	}

	// Gather the set of variable IDs for all secrets that need to be
	// retrieved from Conjur.
	var variableIDs []string
	for key := range updateDests {
		variableIDs = append(variableIDs, key)
	}
	return variableIDs, nil
}

func (p K8sProvider) retrieveConjurSecrets(tracer trace.Tracer) (map[string][]byte, error) {
	spanCtx, span := tracer.Start(p.traceContext, "Fetch Conjur Secrets")
	defer span.End()

	variableIDs, err := p.listConjurSecretsToFetch()
	if err != nil {
		return nil, err
	}

	retrievedConjurSecrets, err := p.conjur.retrieveSecrets(variableIDs, spanCtx)
	if err != nil {
		span.RecordErrorAndSetStatus(err)
		return nil, p.log.recordedError(messages.CSPFK034E, err.Error())
	}
	return retrievedConjurSecrets, nil
}

func (p K8sProvider) updateRequiredK8sSecrets(
	conjurSecrets map[string][]byte, tracer trace.Tracer) (bool, error) {

	var updated bool

	spanCtx, span := tracer.Start(p.traceContext, "Update K8s Secrets")
	defer span.End()

	// Create a map of entries to be added to the 'Data' fields of each
	// K8s Secret. Each entry will map an application secret name to
	// a value retrieved from Conjur.
	newSecretData := map[string]map[string][]byte{}
	for variableID, secretValue := range conjurSecrets {
		dests := p.secretsState.updateDestinations[variableID]
		for _, dest := range dests {
			k8sSecretName := dest.k8sSecretName
			secretName := dest.secretName
			// If there are no data entries for this K8s Secret yet, initialize
			// its map of data entries.
			if newSecretData[k8sSecretName] == nil {
				newSecretData[k8sSecretName] = map[string][]byte{}
			}
			newSecretData[k8sSecretName][secretName] = secretValue
		}
		// Null out the secret value
		conjurSecrets[variableID] = []byte{}
	}

	// Update K8s Secrets with the retrieved Conjur secrets
	for k8sSecretName, secretData := range newSecretData {
		_, childSpan := tracer.Start(spanCtx, "Update K8s Secret")
		defer childSpan.End()
		b := new(bytes.Buffer)
		_, err := fmt.Fprintf(b, "%v", secretData)
		if err != nil {
			p.log.debug(messages.CSPFK005D, err.Error())
			childSpan.RecordErrorAndSetStatus(err)
			return updated, p.log.recordedError(messages.CSPFK022E)
		}

		// Calculate a sha256 checksum on the content
		checksum, _ := utils.FileChecksum(b)

		if utils.ContentHasChanged(k8sSecretName, checksum, p.prevSecretsChecksums) {
			err := p.k8s.updateSecret(
				p.podNamespace,
				k8sSecretName,
				p.secretsState.originalK8sSecrets[k8sSecretName],
				secretData)
			if err != nil {
				// Error messages returned from K8s should be printed only in debug mode
				p.log.debug(messages.CSPFK005D, err.Error())
				childSpan.RecordErrorAndSetStatus(err)
				return false, p.log.recordedError(messages.CSPFK022E)
			}
			p.prevSecretsChecksums[k8sSecretName] = checksum
			updated = true
		} else {
			p.log.info(messages.CSPFK020I)
			updated = false
		}
	}

	return updated, nil
}
