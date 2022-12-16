package k8ssecretsstorage

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
	conjurMocks "github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/clients/conjur/mocks"
	k8sStorageMocks "github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/k8s_secrets_storage/mocks"
)

var testConjurSecrets = map[string]string{
	"conjur/var/path1":        "secret-value1",
	"conjur/var/path2":        "secret-value2",
	"conjur/var/path3":        "secret-value3",
	"conjur/var/path4":        "secret-value4",
	"conjur/var/empty-secret": "",
}

type testMocks struct {
	conjurClient *conjurMocks.ConjurClient
	kubeClient   *k8sStorageMocks.KubeSecretsClient
	logger       *k8sStorageMocks.Logger
}

func newTestMocks() testMocks {
	mocks := testMocks{
		conjurClient: conjurMocks.NewConjurClient(),
		kubeClient:   k8sStorageMocks.NewKubeSecretsClient(),
		logger:       k8sStorageMocks.NewLogger(),
	}
	// Populate Conjur with some test secrets
	mocks.conjurClient.AddSecrets(testConjurSecrets)
	return mocks
}

func (m testMocks) setPermissions(denyConjurRetrieve, denyK8sRetrieve,
	denyK8sUpdate bool) {
	if denyConjurRetrieve {
		m.conjurClient.ErrOnExecute = errors.New("custom error")
	}
	if denyK8sRetrieve {
		m.kubeClient.CanRetrieve = false
	}
	if denyK8sUpdate {
		m.kubeClient.CanUpdate = false
	} else {
		m.kubeClient.CanUpdate = true
	}
}

func (m testMocks) newProvider(requiredSecrets []string) K8sProvider {
	return newProvider(
		k8sProviderDeps{
			k8s: k8sAccessDeps{
				m.kubeClient.RetrieveSecret,
				m.kubeClient.UpdateSecret,
			},
			conjur: conjurAccessDeps{
				m.conjurClient.RetrieveSecrets,
			},
			log: logDeps{
				m.logger.RecordedError,
				m.logger.Error,
				m.logger.Warn,
				m.logger.Info,
				m.logger.Debug,
			},
		},
		true,
		K8sProviderConfig{
			RequiredK8sSecrets: requiredSecrets,
			PodNamespace:       "someNamespace",
		},
		context.Background())
}

type assertFunc func(*testing.T, testMocks, bool, error, string)
type expectedK8sSecrets map[string]map[string]string
type expectedMissingValues map[string][]string

func assertErrorContains(expErrStr string, expectUpdated bool) assertFunc {
	return func(t *testing.T, _ testMocks,
		updated bool, err error, desc string) {

		assert.Error(t, err, desc)
		assert.Contains(t, err.Error(), expErrStr, desc)
		assert.Equal(t, expectUpdated, updated, desc)
	}
}

func assertSecretsUpdated(expK8sSecrets expectedK8sSecrets,
	expMissingValues expectedMissingValues, expectError bool) assertFunc {
	return func(t *testing.T, mocks testMocks, updated bool,
		err error, desc string) {

		if expectError {
			assert.Error(t, err, desc)
		} else {
			assert.NoError(t, err, desc)
			assert.True(t, updated, desc)
		}

		// Check that K8s Secrets contain expected Conjur secret values
		for k8sSecretName, expSecretData := range expK8sSecrets {
			actualSecretData := mocks.kubeClient.InspectSecret(k8sSecretName)
			for secretName, expSecretValue := range expSecretData {
				newDesc := desc + ", Secret: " + secretName
				actualSecretValue := string(actualSecretData[secretName])
				assert.Equal(t, expSecretValue, actualSecretValue, newDesc)
			}
		}
		// Check for secret values leaking into the wrong K8s Secrets
		for k8sSecretName, expMissingValue := range expMissingValues {
			actualSecretData := mocks.kubeClient.InspectSecret(k8sSecretName)
			for _, value := range actualSecretData {
				actualValue := string(value)
				newDesc := desc + ", Leaked secret value: " + actualValue
				assert.NotEqual(t, expMissingValue, actualValue, newDesc)
			}
		}
	}
}

func assertErrorLogged(msg string, args ...interface{}) assertFunc {
	return func(t *testing.T, mocks testMocks, updated bool, err error, desc string) {
		errStr := fmt.Sprintf(msg, args...)
		newDesc := desc + ", error logged: " + errStr
		assert.True(t, mocks.logger.ErrorWasLogged(errStr), newDesc)
	}
}

func TestProvide(t *testing.T) {
	testCases := []struct {
		desc               string
		k8sSecrets         k8sStorageMocks.K8sSecrets
		requiredSecrets    []string
		denyConjurRetrieve bool
		denyK8sRetrieve    bool
		denyK8sUpdate      bool
		asserts            []assertFunc
	}{
		{
			desc: "Happy path, existing k8s Secret with existing Conjur secret",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			asserts: []assertFunc{
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {"secret1": "secret-value1"},
					},
					expectedMissingValues{},
					false,
				),
			},
		},
		{
			desc: "Happy path, 2 existing k8s Secrets with 2 existing Conjur secrets",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {
						"secret1": "conjur/var/path1",
						"secret2": "conjur/var/path2",
					},
				},
				"k8s-secret2": {
					"conjur-map": {
						"secret3": "conjur/var/path3",
						"secret4": "conjur/var/path4",
					},
				},
			},
			requiredSecrets: []string{"k8s-secret1", "k8s-secret2"},
			asserts: []assertFunc{
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {
							"secret1": "secret-value1",
							"secret2": "secret-value2",
						},
						"k8s-secret2": {
							"secret3": "secret-value3",
							"secret4": "secret-value4",
						},
					},
					expectedMissingValues{
						"k8s-secret1": {"secret-value3", "secret-value4"},
						"k8s-secret2": {"secret-value1", "secret-value2"},
					},
					false,
				),
			},
		},
		{
			desc: "Happy path, 2 k8s Secrets use the same Conjur secret with different names",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {
						"secret1": "conjur/var/path1",
						"secret2": "conjur/var/path2",
					},
				},
				"k8s-secret2": {
					"conjur-map": {
						"secret3": "conjur/var/path2",
						"secret4": "conjur/var/path4",
					},
				},
			},
			requiredSecrets: []string{"k8s-secret1", "k8s-secret2"},
			asserts: []assertFunc{
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {
							"secret1": "secret-value1",
							"secret2": "secret-value2",
						},
						"k8s-secret2": {
							"secret3": "secret-value2",
							"secret4": "secret-value4",
						},
					},
					expectedMissingValues{
						"k8s-secret1": {"secret-value4"},
						"k8s-secret2": {"secret-value1"},
					},
					false,
				),
			},
		},
		{
			desc: "Happy path, 2 existing k8s Secrets but only 1 managed by SP",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {
						"secret1": "conjur/var/path1",
						"secret2": "conjur/var/path2",
					},
				},
				"k8s-secret2": {
					"conjur-map": {
						"secret2": "conjur/var/path2",
						"secret3": "conjur/var/path3",
					},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			asserts: []assertFunc{
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {
							"secret1": "secret-value1",
							"secret2": "secret-value2",
						},
					},
					expectedMissingValues{
						"k8s-secret1": {"secret-value3"},
						"k8s-secret2": {"secret-value1", "secret-value2", "secret-value3"},
					},
					false,
				),
			},
		},
		{
			desc: "Happy path, k8s Secret maps to Conjur secret with null string value",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/empty-secret"},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			asserts: []assertFunc{
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {"secret1": ""},
					},
					expectedMissingValues{},
					false,
				),
			},
		},
		{
			desc: "K8s Secrets maps to a non-existent Conjur secret",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "nonexistent/conjur/var"},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			denyK8sRetrieve: true,
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK020E),
				assertErrorContains(messages.CSPFK021E, false),
			},
		},
		{
			desc: "Read access to K8s Secrets is not permitted",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			denyK8sRetrieve: true,
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK020E),
				assertErrorContains(messages.CSPFK021E, false),
			},
		},
		{
			desc: "Access to Conjur secrets is not authorized",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets:    []string{"k8s-secret1"},
			denyConjurRetrieve: true,
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK034E, "custom error"),
				assertErrorContains(fmt.Sprintf(messages.CSPFK034E, "custom error"), false),
			},
		},
		{
			desc: "Updates to K8s 'Secrets' are not permitted",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			denyK8sUpdate:   true,
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK022E),
				assertErrorContains(messages.CSPFK023E, false),
			},
		},
		{
			desc: "K8s secret is required but does not exist",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets: []string{"non-existent-k8s-secret"},
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK020E),
				assertErrorContains(messages.CSPFK021E, false),
			},
		},
		{
			desc: "K8s secret has no 'conjur-map' entry",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"foobar": {"foo": "bar"},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK028E, "k8s-secret1"),
				assertErrorContains(messages.CSPFK021E, false),
			},
		},
		{
			desc: "K8s secret has an empty 'conjur-map' entry",
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {},
				},
			},
			requiredSecrets: []string{"k8s-secret1"},
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK028E, "k8s-secret1"),
				assertErrorContains(messages.CSPFK021E, false),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Set up test case
			mocks := newTestMocks()
			mocks.setPermissions(tc.denyConjurRetrieve, tc.denyK8sRetrieve,
				tc.denyK8sUpdate)
			for secretName, secretData := range tc.k8sSecrets {
				mocks.kubeClient.AddSecret(secretName, secretData)
			}
			provider := mocks.newProvider(tc.requiredSecrets)

			// Run test case
			updated, err := provider.Provide()

			// Confirm results
			for _, assert := range tc.asserts {
				assert(t, mocks, updated, err, tc.desc)
			}
		})
	}
}

func TestSecretsContentChanges(t *testing.T) {

	var desc string
	var k8sSecrets k8sStorageMocks.K8sSecrets
	var requiredSecrets []string
	var denyConjurRetrieve bool
	var denyK8sRetrieve bool
	var denyK8sUpdate bool

	// Initial case, k8s secret should be updated
	desc = "Only update secrets when there are changes"
	k8sSecrets = k8sStorageMocks.K8sSecrets{
		"k8s-secret1": {
			"conjur-map": {"secret1": "conjur/var/path1"},
		},
	}
	requiredSecrets = []string{"k8s-secret1"}
	mocks := newTestMocks()
	mocks.setPermissions(denyConjurRetrieve, denyK8sRetrieve, denyK8sUpdate)
	for secretName, secretData := range k8sSecrets {
		mocks.kubeClient.AddSecret(secretName, secretData)
	}
	provider := mocks.newProvider(requiredSecrets)
	update, err := provider.Provide()
	assert.False(t, mocks.logger.InfoWasLogged(messages.CSPFK020I))
	assertSecretsUpdated(
		expectedK8sSecrets{
			"k8s-secret1": {"secret1": "secret-value1"},
		},
		expectedMissingValues{}, false)(t, mocks, update, err, desc)

	// Call Provide again, verify it doesn't try to update the secret
	// as there should be an error if it tried to write the secrets
	desc = "Verify secrets are not updated when there are no changes"
	denyK8sUpdate = true
	mocks.setPermissions(denyConjurRetrieve, denyK8sRetrieve, denyK8sUpdate)
	update, err = provider.Provide()
	assert.NoError(t, err)
	assert.True(t, mocks.logger.InfoWasLogged(messages.CSPFK020I))
	// verify the same secret still exists
	assertSecretsUpdated(
		expectedK8sSecrets{
			"k8s-secret1": {"secret1": "secret-value1"},
		},
		expectedMissingValues{}, false)(t, mocks, true, err, desc)

	// Change the k8s secret and verify a new secret is written
	desc = "Verify new secrets are written when there are changes to the Conjur secret"
	mocks.logger.ClearInfo()
	secrets, _ := mocks.kubeClient.RetrieveSecret("", "k8s-secret1")
	var newMap = map[string][]byte{
		"conjur-map": []byte("secret2: conjur/var/path2"),
	}
	denyK8sUpdate = false
	mocks.setPermissions(denyConjurRetrieve, denyK8sRetrieve, denyK8sUpdate)
	mocks.kubeClient.UpdateSecret("mock namespace", "k8s-secret1", secrets, newMap)
	update, err = provider.Provide()
	assert.NoError(t, err)
	assertSecretsUpdated(
		expectedK8sSecrets{
			"k8s-secret1": {"secret2": "secret-value2"},
		},
		expectedMissingValues{}, false)(t, mocks, update, err, desc)
	assert.False(t, mocks.logger.InfoWasLogged(messages.CSPFK020I))

	// call again with no changes
	desc = "Verify again secrets are not updated when there are no changes"
	update, err = provider.Provide()
	assert.NoError(t, err)
	assert.True(t, mocks.logger.InfoWasLogged(messages.CSPFK020I))

	// verify a new k8s secret is written when the Conjur secret changes
	desc = "Verify new secrets are written when there are changes to the k8s secret"
	mocks.logger.ClearInfo()
	var updateConjurSecrets = map[string]string{
		"conjur/var/path1":        "new-secret-value1",
		"conjur/var/path2":        "new-secret-value2",
		"conjur/var/path3":        "new-secret-value3",
		"conjur/var/path4":        "new-secret-value4",
		"conjur/var/empty-secret": "",
	}
	mocks.conjurClient.AddSecrets(updateConjurSecrets)
	update, err = provider.Provide()
	assert.False(t, mocks.logger.InfoWasLogged(messages.CSPFK020I))
	assertSecretsUpdated(
		expectedK8sSecrets{
			"k8s-secret1": {"secret2": "new-secret-value2"},
		},
		expectedMissingValues{}, false)(t, mocks, update, err, desc)
}

func TestProvideSanitization(t *testing.T) {
	testCases := []struct {
		desc             string
		k8sSecrets       k8sStorageMocks.K8sSecrets
		requiredSecrets  []string
		sanitizeEnabled  bool
		retrieveErrorMsg string
		asserts          []assertFunc
	}{
		{
			desc:            "403 error",
			sanitizeEnabled: true,
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets:  []string{"k8s-secret1"},
			retrieveErrorMsg: "403",
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK034E, "403"),
				assertErrorContains(fmt.Sprintf(messages.CSPFK034E, "403"), true),
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {"secret1": ""},
					},
					expectedMissingValues{},
					true,
				),
			},
		},
		{
			desc:            "404 error",
			sanitizeEnabled: true,
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets:  []string{"k8s-secret1"},
			retrieveErrorMsg: "404",
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK034E, "404"),
				assertErrorContains(fmt.Sprintf(messages.CSPFK034E, "404"), true),
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {"secret1": ""},
					},
					expectedMissingValues{},
					true,
				),
			},
		},
		{
			desc:            "generic error doesn't delete secret",
			sanitizeEnabled: true,
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets:  []string{"k8s-secret1"},
			retrieveErrorMsg: "generic error",
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK034E, "generic error"),
				assertErrorContains(fmt.Sprintf(messages.CSPFK034E, "generic error"), false),
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {"secret1": "secret-value1"},
					},
					expectedMissingValues{},
					true,
				),
			},
		},
		{
			desc:            "403 error with sanitize disabled",
			sanitizeEnabled: false,
			k8sSecrets: k8sStorageMocks.K8sSecrets{
				"k8s-secret1": {
					"conjur-map": {"secret1": "conjur/var/path1"},
				},
			},
			requiredSecrets:  []string{"k8s-secret1"},
			retrieveErrorMsg: "403",
			asserts: []assertFunc{
				assertErrorLogged(messages.CSPFK034E, "403"),
				assertErrorContains(fmt.Sprintf(messages.CSPFK034E, "403"), false),
				assertSecretsUpdated(
					expectedK8sSecrets{
						"k8s-secret1": {"secret1": "secret-value1"},
					},
					expectedMissingValues{},
					true,
				),
			},
		},
	}

	for _, tc := range testCases {
		// Set up test case
		mocks := newTestMocks()

		// First do a clean run will all permissions allowed to retrieve and populate the K8s secrets
		provider := mocks.newProvider(tc.requiredSecrets)
		provider.sanitizeEnabled = tc.sanitizeEnabled
		for secretName, secretData := range tc.k8sSecrets {
			mocks.kubeClient.AddSecret(secretName, secretData)
		}
		updated, err := provider.Provide()
		assert.NoError(t, err, tc.desc)
		assert.True(t, updated)

		// Now run test case, injecting an error into the retrieve function
		mocks.conjurClient.ErrOnExecute = errors.New(tc.retrieveErrorMsg)
		updated, err = provider.Provide()

		// Confirm results
		for _, assert := range tc.asserts {
			assert(t, mocks, updated, err, tc.desc)
		}
	}
}
