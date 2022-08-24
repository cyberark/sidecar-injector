package inject

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestSecretsProviderSidecarInjection(t *testing.T) {
	var testCases = []injectionTestCase{
		{
			description:                         "SecretsProvider sidecar",
			annotatedPodTemplateSpecPath:        "./testdata/secrets-provider-annotated-pod.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/secrets-provider-mutated-pod.json",
			env: map[string]string{
				"CONJUR_ACCOUNT":          "myConjurAccount",
				"CONJUR_APPLIANCE_URL":    "https://conjur-oss.conjur-oss.svc.cluster.local",
				"CONJUR_AUTHENTICATOR_ID": "my-authenticator-id",
				"CONJUR_AUTHN_URL":        "https://conjur-oss.conjur-oss.svc.cluster.local/authn-k8s/my-authenticator-id",
				"CONJUR_SSL_CERTIFICATE":  "-----BEGIN CERTIFICATE-----tVw0ZnjsOV2ZeIBRalX/72RplPzkmWKAw==\n-----END CERTIFICATE-----\n",
			},
		},
		{
			description:                         "SecretsProvider init",
			annotatedPodTemplateSpecPath:        "./testdata/secrets-provider-init-annotated-pod.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/secrets-provider-init-mutated-pod.json",
			env: map[string]string{
				"CONJUR_ACCOUNT":          "myConjurAccount",
				"CONJUR_APPLIANCE_URL":    "https://conjur-oss.conjur-oss.svc.cluster.local",
				"CONJUR_AUTHENTICATOR_ID": "my-authenticator-id",
				"CONJUR_AUTHN_URL":        "https://conjur-oss.conjur-oss.svc.cluster.local/authn-k8s/my-authenticator-id",
				"CONJUR_SSL_CERTIFICATE":  "-----BEGIN CERTIFICATE-----tVw0ZnjsOV2ZeIBRalX/72RplPzkmWKAw==\n-----END CERTIFICATE-----\n",
			},
		},
		{
			description:                         "SecretsProvider golden config",
			annotatedPodTemplateSpecPath:        "./testdata/secrets-provider-annotated-pod.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/secrets-provider-mutated-pod.json",
			env: map[string]string{
				"conjurAccount":           "myConjurAccount",
				"conjurApplianceUrl":      "https://conjur-oss.conjur-oss.svc.cluster.local",
				"authnK8sAuthenticatorID": "my-authenticator-id",
				"conjurSslCertificate":    "-----BEGIN CERTIFICATE-----tVw0ZnjsOV2ZeIBRalX/72RplPzkmWKAw==\n-----END CERTIFICATE-----\n",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			for envVar, value := range tc.env {
				os.Setenv(envVar, value)
			}
			// Create the Admission Request (wrapped in an Admission Review) from the
			// annotated pod fixture. The goal is to use the annotations as a signal to
			// the sidecar-injector to mutate the Pod template spec.
			req, err := newTestAdmissionRequest(
				tc.annotatedPodTemplateSpecPath,
			)
			if !assert.NoError(t, err) {
				return
			}

			// Read the expected Pod template spec fixture.
			expectedMod, err := ioutil.ReadFile(
				tc.expectedInjectedPodTemplateSpecPath,
			)
			if !assert.NoError(t, err) {
				return
			}

			// Get the modified Pod template spec from the input.
			mod, err := applyPatchToAdmissionRequest(req)
			if !assert.NoError(t, err) {
				return
			}

			// Assert that the modified Pod template spec should equal the expected Pod
			// template spec.
			assert.JSONEq(t, string(expectedMod), string(mod))
			for envVar, _ := range tc.env {
				os.Unsetenv(envVar)
			}
		})
	}
}
