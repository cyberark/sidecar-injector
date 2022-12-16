package conjur

import (
	"context"
	"fmt"
	"strings"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/access_token/memory"
	"github.com/cyberark/conjur-authn-k8s-client/pkg/authenticator"
	"github.com/cyberark/conjur-authn-k8s-client/pkg/authenticator/config"
	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/cyberark/conjur-opentelemetry-tracer/pkg/trace"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

// SecretRetriever implements a Retrieve function that is capable of
// authenticating with Conjur and retrieving multiple Conjur variables
// in bulk.
type SecretRetriever struct {
	authn authenticator.Authenticator
}

// RetrieveSecretsFunc defines a function type for retrieving secrets.
type RetrieveSecretsFunc func(variableIDs []string, traceContext context.Context) (map[string][]byte, error)

// NewSecretRetriever creates a new SecretRetriever and Authenticator
// given an authenticator config.
func NewSecretRetriever(authnConfig config.Configuration) (*SecretRetriever, error) {
	accessToken, err := memory.NewAccessToken()
	if err != nil {
		return nil, fmt.Errorf("%s", messages.CSPFK001E)
	}

	authn, err := authenticator.NewAuthenticatorWithAccessToken(authnConfig, accessToken)
	if err != nil {
		return nil, fmt.Errorf("%s", messages.CSPFK009E)
	}

	return &SecretRetriever{
		authn: authn,
	}, nil
}

// Retrieve implements a RetrieveSecretsFunc for a given SecretRetriever.
// Authenticates the client, and retrieves a given batch of variables from Conjur.
func (retriever SecretRetriever) Retrieve(variableIDs []string, traceContext context.Context) (map[string][]byte, error) {

	authn := retriever.authn

	err := authn.AuthenticateWithContext(traceContext)
	if err != nil {
		return nil, log.RecordedError(messages.CSPFK010E)
	}

	accessTokenData, err := authn.GetAccessToken().Read()
	if err != nil {
		return nil, log.RecordedError(messages.CSPFK002E)
	}
	// Always delete the access token. The deletion is idempotent and never fails
	defer authn.GetAccessToken().Delete()

	tr := trace.NewOtelTracer(otel.Tracer("secrets-provider"))
	_, span := tr.Start(traceContext, "Retrieve secrets")
	span.SetAttributes(attribute.Int("variable_count", len(variableIDs)))
	defer span.End()

	return retrieveConjurSecrets(accessTokenData, variableIDs)
}

func retrieveConjurSecrets(accessToken []byte, variableIDs []string) (map[string][]byte, error) {
	log.Info(messages.CSPFK003I, variableIDs)

	if len(variableIDs) == 0 {
		log.Info(messages.CSPFK016I)
		return nil, nil
	}

	conjurClient, err := NewConjurClient(accessToken)
	if err != nil {
		return nil, log.RecordedError(messages.CSPFK033E)
	}

	retrievedSecretsByFullIDs, err := conjurClient.RetrieveBatchSecrets(variableIDs)
	if err != nil {
		return nil, err
	}

	// Normalise secret IDs from batch secrets back to <variable_id>
	var retrievedSecrets = map[string][]byte{}
	for id, secret := range retrievedSecretsByFullIDs {
		retrievedSecrets[normaliseVariableId(id)] = secret
		delete(retrievedSecretsByFullIDs, id)
	}

	return retrievedSecrets, nil
}

// The variable ID can be in the format "<account>:variable:<variable_id>". This function
// just makes sure that if a variable is of the form "<account>:variable:<variable_id>"
// we normalise it to "<variable_id>", otherwise we just leave it be!
func normaliseVariableId(fullVariableId string) string {
	variableIdParts := strings.SplitN(fullVariableId, ":", 3)
	if len(variableIdParts) == 3 {
		return variableIdParts[2]
	}

	return fullVariableId
}
