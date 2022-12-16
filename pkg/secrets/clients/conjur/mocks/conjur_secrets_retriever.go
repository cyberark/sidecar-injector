package mocks

import (
	"context"
	"errors"
)

/*
	Determines if Conjur secrets have 'execute' privileges by mapping `execute` to true or false. We don't
	dive deeper into the granularity at the level of each Conjur variable because for this suite of tests we
	aren't concerned with if some Conjur secrets have permissions and others don't. Our main priority is
	validating that those secrets with 'execute' permissions can be fetched.
*/

type ConjurClient struct {
	ErrOnExecute error
	Database     map[string]string
}

func (c *ConjurClient) RetrieveSecrets(secretIds []string, ctx context.Context) (map[string][]byte, error) {
	res := make(map[string][]byte)

	if c.ErrOnExecute != nil {
		return nil, c.ErrOnExecute
	}

	for _, secretId := range secretIds {
		// Check if the secret exists in the mock Conjur DB
		variableData, ok := c.Database[secretId]
		if !ok {
			return nil, errors.New("no_conjur_secret_error")
		}

		res[secretId] = []byte(variableData)
	}

	return res, nil
}

func NewConjurClient() *ConjurClient {
	database := map[string]string{
		"conjur_variable1":             "conjur_secret1",
		"conjur_variable2":             "conjur_secret2",
		"conjur_variable_empty_secret": "",
	}

	return &ConjurClient{
		ErrOnExecute: nil,
		Database:     database,
	}
}

func (c *ConjurClient) AddSecrets(
	secrets map[string]string,
) {
	for id, secret := range secrets {
		c.Database[id] = secret
	}
}
