package pushtofile

import (
	"context"
	"os"
	"strings"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"
	"github.com/cyberark/conjur-opentelemetry-tracer/pkg/trace"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/clients/conjur"
	"go.opentelemetry.io/otel"
)

type fileProvider struct {
	retrieveSecretsFunc conjur.RetrieveSecretsFunc
	secretGroups        []*SecretGroup
	traceContext        context.Context
	sanitizeEnabled     bool
}

type fileProviderDepFuncs struct {
	retrieveSecretsFunc conjur.RetrieveSecretsFunc
	depOpenWriteCloser  openWriteCloserFunc
	depPushToWriter     pushToWriterFunc
}

// P2FProviderConfig provides config specific to Push-to-File provider
type P2FProviderConfig struct {
	SecretFileBasePath   string
	TemplateFileBasePath string
	AnnotationsMap       map[string]string
}

// NewProvider creates a new provider for Push-to-File mode.
func NewProvider(
	retrieveSecretsFunc conjur.RetrieveSecretsFunc,
	sanitizeEnabled bool,
	config P2FProviderConfig,
) (*fileProvider, []error) {

	secretGroups, err := NewSecretGroups(config.SecretFileBasePath,
		config.TemplateFileBasePath, config.AnnotationsMap)
	if err != nil {
		return nil, err
	}

	return &fileProvider{
		retrieveSecretsFunc: retrieveSecretsFunc,
		secretGroups:        secretGroups,
		traceContext:        nil,
		sanitizeEnabled:     sanitizeEnabled,
	}, nil
}

// Provide implements a ProviderFunc to retrieve and push secrets to the filesystem.
func (p fileProvider) Provide() (bool, error) {
	return provideWithDeps(
		p.traceContext,
		p.secretGroups,
		p.sanitizeEnabled,
		fileProviderDepFuncs{
			retrieveSecretsFunc: p.retrieveSecretsFunc,
			depOpenWriteCloser:  openFileAsWriteCloser,
			depPushToWriter:     pushToWriter,
		},
	)
}

func (p *fileProvider) SetTraceContext(ctx context.Context) {
	p.traceContext = ctx
}

func provideWithDeps(
	traceContext context.Context,
	groups []*SecretGroup,
	sanitizeEnabled bool,
	depFuncs fileProviderDepFuncs,
) (bool, error) {
	// Use the global TracerProvider
	tr := trace.NewOtelTracer(otel.Tracer("secrets-provider"))
	spanCtx, span := tr.Start(traceContext, "Fetch Conjur Secrets")
	var updated bool
	secretsByGroup, err := FetchSecretsForGroups(depFuncs.retrieveSecretsFunc, groups, spanCtx)
	if err != nil {
		// Delete secret files for variables that no longer exist or the user no longer has permissions to.
		// In the future we'll delete only the secrets that are revoked, but for now we delete all secrets in
		// the group because we don't have a way to determine which secrets are revoked.
		if (strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "404")) && sanitizeEnabled {
			updated = true
			for _, group := range groups {
				log.Info(messages.CSPFK019I)
				rmErr := os.Remove(group.FilePath)
				if rmErr != nil && !os.IsNotExist(rmErr) {
					log.Error(messages.CSPFK062E, rmErr)
				}
			}
		}

		span.RecordErrorAndSetStatus(err)
		span.End()
		return updated, err
	}
	span.End()

	spanCtx, span = tr.Start(traceContext, "Write Secret Files")
	defer span.End()
	for _, group := range groups {
		_, childSpan := tr.Start(spanCtx, "Write Secret Files for group")
		defer childSpan.End()
		groupUpdated, err := group.pushToFileWithDeps(
			depFuncs.depOpenWriteCloser,
			depFuncs.depPushToWriter,
			secretsByGroup[group.Name],
		)
		if err != nil {
			childSpan.RecordErrorAndSetStatus(err)
			span.RecordErrorAndSetStatus(err)
			return updated, err
		}
		if groupUpdated {
			updated = true
		}
	}

	log.Info(messages.CSPFK015I)
	return updated, nil
}
