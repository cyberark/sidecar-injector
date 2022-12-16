package mocks

import (
	"errors"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
)

// K8sSecrets represents a collection of Kubernetes Secrets to be populated
// into the mock Kubernetes client's database. The logical hierarchy
// represented by this structure is:
// - Each Kubernetes Secret contains a 'Data' field.
// - Each 'Data' field contains one or more entries that are key/value pairs.
// - The value in each 'Data' field entry can be a nested set of
//   key/value pairs. In particular, for the entry with the key
//   'conjur-info', the value is expected to be a mapping of application
//   secret names to the corresponding Conjur variable ID (or policy path)
//   that should be used to retrieve the secret value.
type K8sSecrets map[string]k8sSecretData
type k8sSecretData map[string]k8sSecretDataValues
type k8sSecretDataValues map[string]string

// KubeSecretsClient implements a mock Kubernetes client for testing
// Kubernetes Secrets access by the Secrets Provider. This client provides:
// - A Kubernetes Secret retrieve function
// - A Kubernetes Secret update function
// Kubernetes Secrets are populated for this mock client via the
// AddSecret method. Retrieval and update errors can be simulated
// for testing by setting the 'CanRetrieve' and 'CanUpdate' flags
// (respectively) to false.
type KubeSecretsClient struct {
	// Mocks a K8s database. Maps k8s secret names to K8s secrets.
	database map[string]map[string][]byte
	// TODO: CanRetrieve and CanUpdate are really just used to assert on the presence of errors
	// 	and should probably just be an optional error.
	CanRetrieve bool
	CanUpdate   bool
}

// NewKubeSecretsClient creates an instance of a KubeSecretsClient
func NewKubeSecretsClient() *KubeSecretsClient {
	client := KubeSecretsClient{
		database:    map[string]map[string][]byte{},
		CanRetrieve: true,
		CanUpdate:   true,
	}

	return &client
}

// AddSecret adds a Kubernetes Secret to the mock Kubernetes Secrets client's
// database.
func (c *KubeSecretsClient) AddSecret(
	secretName string,
	secretData k8sSecretData,
) {
	// Convert string values to YAML format
	yamlizedSecretData := map[string][]byte{}
	for key, value := range secretData {
		yamlValue, err := yaml.Marshal(value)
		if err != nil {
			panic(err)
		}
		yamlizedSecretData[key] = yamlValue
	}

	c.database[secretName] = yamlizedSecretData
}

// RetrieveSecret retrieves a Kubernetes Secret from the mock Kubernetes
// Secrets client's database.
func (c *KubeSecretsClient) RetrieveSecret(_ string, secretName string) (*v1.Secret, error) {

	if !c.CanRetrieve {
		return nil, errors.New("custom error")
	}

	// Check if the secret exists in the mock K8s DB
	secretData, ok := c.database[secretName]
	if !ok {
		return nil, errors.New("custom error")
	}

	return &v1.Secret{
		Data: secretData,
	}, nil
}

// UpdateSecret updates a Kubernetes Secret in the mock Kubernetes
// Secrets client's database.
func (c *KubeSecretsClient) UpdateSecret(
	_ string, secretName string,
	originalK8sSecret *v1.Secret,
	stringDataEntriesMap map[string][]byte) error {

	if !c.CanUpdate {
		return errors.New("custom error")
	}

	secretToUpdate := c.database[secretName]
	for key, value := range stringDataEntriesMap {
		secretToUpdate[key] = value
	}

	return nil
}

// InspectSecret provides a way for unit tests to view the 'Data' field
// content of a Kubernetes Secret by reading this content directly from
// the mock client's database.
func (c *KubeSecretsClient) InspectSecret(secretName string) map[string][]byte {
	return c.database[secretName]
}
