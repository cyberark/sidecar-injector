package conjur

import (
	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

/*
	Client for communication with Conjur. In this project it is used only for
    batch secrets retrieval so we expose only this method of the client.

	The name ConjurClient also improves readability as Client can be ambiguous.
*/
type ConjurClient interface {
	RetrieveBatchSecrets([]string) (map[string][]byte, error)
}

func NewConjurClient(tokenData []byte) (ConjurClient, error) {
	log.Info(messages.CSPFK002I)
	config, err := conjurapi.LoadConfig()
	if err != nil {
		return nil, log.RecordedError(messages.CSPFK031E, err.Error())
	}

	client, err := conjurapi.NewClientFromToken(config, string(tokenData))
	if err != nil {
		return nil, log.RecordedError(messages.CSPFK032E, err.Error())
	}

	return client, nil
}
