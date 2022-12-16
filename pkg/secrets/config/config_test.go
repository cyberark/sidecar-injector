package config

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

type errorAssertFunc func(*testing.T, []error, []error)

func assertEmptyErrorList() errorAssertFunc {
	return func(t *testing.T, errorList []error, infoList []error) {
		assert.Empty(t, errorList)
	}
}

func assertInfoInList(err error) errorAssertFunc {
	return func(t *testing.T, errorList []error, infoList []error) {
		assert.Contains(t, infoList, err)
	}
}

func assertErrorInList(err error) errorAssertFunc {
	return func(t *testing.T, errorList []error, infoList []error) {
		assert.Contains(t, errorList, err)
	}
}

func assertGoodMap(expected map[string]string) func(*testing.T, map[string]string) {
	return func(t *testing.T, result map[string]string) {
		assert.Equal(t, expected, result)
	}
}

func assertGoodConfig(expected *Config) func(*testing.T, *Config) {
	return func(t *testing.T, result *Config) {
		assert.Equal(t, expected, result)
	}
}

type validateAnnotationsTestCase struct {
	description string
	annotations map[string]string
	assert      errorAssertFunc
}

var validateAnnotationsTestCases = []validateAnnotationsTestCase{
	{
		description: "given properly formatted annotations, no error or info logs are returned",
		annotations: map[string]string{
			AuthnIdentityKey:                           "host/conjur/authn-k8s/cluster/apps/inventory-api",
			"conjur.org/container-mode":                "init",
			"conjur.org/secret-destination":            "file",
			k8sSecretsKey:                              "- secret-1\n- secret-2\n- secret-3\n",
			retryCountLimitKey:                         "12",
			retryIntervalSecKey:                        "2",
			"conjur.org/secrets-refresh-interval":      "5s",
			"conjur.org/conjur-secrets.this-group":     "- test/url\n- test-password: test/password\n- test-username: test/username\n",
			"conjur.org/secret-file-path.this-group":   "this-relative-path",
			"conjur.org/secret-file-format.this-group": "yaml",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "if an annotation does not have the 'conjur.org/' prefix, it is ignored",
		annotations: map[string]string{
			"no-prefix": "some-value",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "if an annotation has the 'conjur.org/' prefix, but is not a supported annotation, an info-level error is returned",
		annotations: map[string]string{
			"conjur.org/valid-but-unrecognized": "def",
		},
		assert: assertInfoInList(fmt.Errorf(messages.CSPFK011I, "conjur.org/valid-but-unrecognized")),
	},
	{
		description: "if an annotation is configured with an invalid value, an error is returned",
		annotations: map[string]string{
			SecretsDestinationKey: "invalid",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK043E, SecretsDestinationKey, "invalid", []string{"file", "k8s_secrets"})),
	},
	{
		description: "when an annotation expects an integer but is given a non-integer value, an error is returned",
		annotations: map[string]string{
			retryCountLimitKey: "seven",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK042E, retryCountLimitKey, "seven", "Integer")),
	},
	{
		description: "when an annotation expects a bool but is given a non-bool value, an error is returned",
		annotations: map[string]string{
			RemoveDeletedSecretsKey: "10",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK042E, RemoveDeletedSecretsKey, "10", "Boolean")),
	},
	{
		description: "when an annotation expects a boolean but is given a non-boolean value, an error is returned",
		annotations: map[string]string{
			debugLoggingKey: "not-a-boolean",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK042E, debugLoggingKey, "not-a-boolean", "Boolean")),
	},
}

type gatherSecretsProviderSettingsTestCase struct {
	description string
	annotations map[string]string
	env         map[string]string
	assert      func(t *testing.T, result map[string]string)
}

var gatherSecretsProviderSettingsTestCases = []gatherSecretsProviderSettingsTestCase{
	{
		description: "the resulting map will those annotations and envvars pertaining to Secrets Provider config",
		annotations: map[string]string{
			SecretsDestinationKey:       "file",
			"conjur.org/container-mode": "init",
		},
		env: map[string]string{
			"SECRETS_DESTINATION": "file",
			"RETRY_COUNT_LIMIT":   "5",
			"UNRELATED_ENVVAR":    "UNRELATED",
		},
		assert: assertGoodMap(map[string]string{
			SecretsDestinationKey:       "file",
			"conjur.org/container-mode": "init",
			"SECRETS_DESTINATION":       "file",
			"RETRY_COUNT_LIMIT":         "5",
		}),
	},
	{
		description: "given an empty annotations map, the returned map should contain the environment",
		annotations: map[string]string{},
		env: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			"SECRETS_DESTINATION": "file",
			"K8S_SECRETS":         "secret-1,secret-2,secret-3",
			"RETRY_COUNT_LIMIT":   "5",
			"RETRY_INTERVAL_SEC":  "12",
		},
		assert: assertGoodMap(map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			"SECRETS_DESTINATION": "file",
			"K8S_SECRETS":         "secret-1,secret-2,secret-3",
			"RETRY_COUNT_LIMIT":   "5",
			"RETRY_INTERVAL_SEC":  "12",
		}),
	},
	{
		description: "given an empty environment, the returned map should contain the annotations",
		annotations: map[string]string{
			SecretsDestinationKey:       "file",
			"conjur.org/container-mode": "init",
		},
		env: map[string]string{},
		assert: assertGoodMap(map[string]string{
			SecretsDestinationKey:       "file",
			"conjur.org/container-mode": "init",
		}),
	},
	{
		description: "annotations and envvars not related to Secrets Provider config are omitted",
		annotations: map[string]string{
			SecretsDestinationKey:        "file",
			"conjur.org/container-mode":  "init",
			"conjur.org/unrelated-annot": "unrelated-value",
		},
		env: map[string]string{
			"MY_POD_NAMESPACE":  "test-namespace",
			"RETRY_COUNT_LIMIT": "5",
			"UNRELATED_ENVVAR":  "unrelated-value",
		},
		assert: assertGoodMap(map[string]string{
			SecretsDestinationKey:       "file",
			"conjur.org/container-mode": "init",
			"MY_POD_NAMESPACE":          "test-namespace",
			"RETRY_COUNT_LIMIT":         "5",
		}),
	},
}

type validateSecretsProviderSettingsTestCase struct {
	description  string
	envAndAnnots map[string]string
	assert       func(t *testing.T, errorResults []error, infoResults []error)
}

var validateSecretsProviderSettingsTestCases = []validateSecretsProviderSettingsTestCase{
	{
		description: "given a valid configuration of annotations, no errors are returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":                    "test-namespace",
			SecretsDestinationKey:                 "file",
			retryCountLimitKey:                    "10",
			retryIntervalSecKey:                   "20",
			k8sSecretsKey:                         "- secret-1\n- secret-2\n- secret-3\n",
			RemoveDeletedSecretsKey:               "true",
			"conjur.org/container-mode":           "sidecar",
			"conjur.org/secrets-refresh-interval": "5m",
			"conjur.org/secrets-refresh-enabled":  "true",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "given a valid configuration of envVars, no errors are returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			"SECRETS_DESTINATION":    "k8s_secrets",
			"RETRY_COUNT_LIMIT":      "10",
			"RETRY_INTERVAL_SEC":     "20",
			"K8S_SECRETS":            "secret-1,secret-2,secret-3",
			"REMOVE_DELETED_SECRETS": "false",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "mixed-source config",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			SecretsDestinationKey: "k8s_secrets",
			"RETRY_COUNT_LIMIT":   "10",
			retryIntervalSecKey:   "20",
			"K8S_SECRETS":         "secret-1,secret-2,secret-3",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "if StoreType is configured with both its annotation and envVar, an info-level error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			"SECRETS_DESTINATION": "k8s_secrets",
			SecretsDestinationKey: "file",
		},
		assert: assertInfoInList(fmt.Errorf(messages.CSPFK012I, "StoreType", "SECRETS_DESTINATION", SecretsDestinationKey)),
	},
	{
		description: "if RequiredK8sSecrets is configured with both its annotation and envVar, an info-level error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			SecretsDestinationKey: "k8s_secrets",
			k8sSecretsKey:         "- secret-1\n- secret-2\n",
			"K8S_SECRETS":         "another-secret-1,another-secret-2",
		},
		assert: assertInfoInList(fmt.Errorf(messages.CSPFK012I, "RequiredK8sSecrets", "K8S_SECRETS", k8sSecretsKey)),
	},
	{
		description: "if RetryCountLimit is configured with both its annotation and envVar, an info-level error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			SecretsDestinationKey: "file",
			retryCountLimitKey:    "10",
			"RETRY_COUNT_LIMIT":   "12",
		},
		assert: assertInfoInList(fmt.Errorf(messages.CSPFK012I, "RetryCountLimit", "RETRY_COUNT_LIMIT", retryCountLimitKey)),
	},
	{
		description: "if RetryIntervalSec is configured with both its annotation and envVar, an info-level error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			SecretsDestinationKey: "file",
			retryIntervalSecKey:   "2",
			"RETRY_INTERVAL_SEC":  "7",
		},
		assert: assertInfoInList(fmt.Errorf(messages.CSPFK012I, "RetryIntervalSec", "RETRY_INTERVAL_SEC", retryIntervalSecKey)),
	},
	{
		description:  "if MY_POD_NAMESPACE envVar is not set, an error is returned",
		envAndAnnots: map[string]string{},
		assert:       assertErrorInList(fmt.Errorf(messages.CSPFK004E, "MY_POD_NAMESPACE")),
	},
	{
		description: "if storeType is not provided by either annotation or envVar, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE": "test-namespace",
		},
		assert: assertErrorInList(errors.New(messages.CSPFK046E)),
	},
	{
		description: "if envVars are used to configure Push-to-File mode, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			"SECRETS_DESTINATION": "file",
		},
		assert: assertErrorInList(errors.New(messages.CSPFK047E)),
	},
	{
		description: "if 'conjur.org/secrets-destination' is provided and malformed, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			SecretsDestinationKey: "invalid",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK043E, SecretsDestinationKey, "invalid", []string{File, K8s})),
	},
	{
		description: "if RequiredK8sSecrets is not configured in K8s Secrets mode, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			SecretsDestinationKey: "k8s_secrets",
		},
		assert: assertErrorInList(errors.New(messages.CSPFK048E)),
	},
	{
		description: "if RequiredK8sSecrets is set to a null string in K8s Secrets mode, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			"SECRETS_DESTINATION": "k8s_secrets",
			"K8S_SECRETS":         "",
		},
		assert: assertErrorInList(errors.New(messages.CSPFK048E)),
	},
	{
		description: "if envVar 'SECRETS_DESTINATION' is malformed in the absence of annotation 'conjur.org/secrets-destination', an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":    "test-namespace",
			"SECRETS_DESTINATION": "invalid",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK005E, "SECRETS_DESTINATION")),
	},
	{
		description: "if refresh interval is malformed, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "5",
			ContainerModeKey:          "sidecar",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK050E, "5", "time: missing unit in duration \"5\"")),
	},
	{
		description: "if refresh interval is malformed with enable set to true, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshEnabledKey:  "true",
			SecretsRefreshIntervalKey: "5",
			ContainerModeKey:          "sidecar",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK050E, "5", "time: missing unit in duration \"5\"")),
	},
	{
		description: "if refresh enable is true and interval not set, no errors are returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			SecretsDestinationKey:    "file",
			SecretsRefreshEnabledKey: "true",
			ContainerModeKey:         "sidecar",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "if refresh enable is true container mode set with env, no errors are returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			SecretsDestinationKey:    "file",
			SecretsRefreshEnabledKey: "true",
			"CONTAINER_MODE":         "sidecar",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "if refresh enable is false and interval not set, no errors are returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			SecretsDestinationKey:    "file",
			SecretsRefreshEnabledKey: "false",
			ContainerModeKey:         "sidecar",
		},
		assert: assertEmptyErrorList(),
	},
	{
		description: "if refresh interval is zero, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "0",
			ContainerModeKey:          "sidecar",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK050E, "0", "Secrets refresh interval must be at least one second")),
	},
	{
		description: "if refresh interval is too small, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "500ms",
			ContainerModeKey:          "sidecar",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK050E, "500ms", "Secrets refresh interval must be at least one second")),
	},
	{
		description: "if refresh interval is negative, an error is returned",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "-5s",
			ContainerModeKey:          "sidecar",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK050E, "-5s", "Secrets refresh interval must be at least one second")),
	},
	{
		description: "if refresh interval is set and enable is false",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "5s",
			SecretsRefreshEnabledKey:  "false",
			ContainerModeKey:          "sidecar",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK050E, "5s", "Secrets refresh interval set to value while enable is false")),
	},
	{
		description: "if refresh interval is set and container mode is init",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "5s",
			ContainerModeKey:          "init",
		},
		assert: assertErrorInList(fmt.Errorf(messages.CSPFK051E, "Secrets refresh is enabled while container mode is set to", "init")),
	},
	{
		description: "if refresh interval is set and container mode set with env",
		envAndAnnots: map[string]string{
			"MY_POD_NAMESPACE":        "test-namespace",
			SecretsRefreshIntervalKey: "5s",
			ContainerModeKey:          "",
			"CONTAINER_MODE":          "sidecar",
			"SECRETS_DESTINATION":     "k8s_secrets",
			"K8S_SECRETS":             "another-secret-1,another-secret-2",
		},
		assert:  assertEmptyErrorList(),
	},
}

type newConfigTestCase struct {
	description string
	settings    map[string]string
	assert      func(t *testing.T, config *Config)
}

var newConfigTestCases = []newConfigTestCase{
	{
		description: "a valid map of annotation-based Secrets Provider settings returns a valid Config",
		settings: map[string]string{
			"MY_POD_NAMESPACE":      "test-namespace",
			SecretsDestinationKey:   "k8s_secrets",
			k8sSecretsKey:           "- secret-1\n- secret-2\n- secret-3\n",
			retryCountLimitKey:      "10",
			retryIntervalSecKey:     "20",
			RemoveDeletedSecretsKey: "false",
		},
		assert: assertGoodConfig(&Config{
			PodNamespace:       "test-namespace",
			StoreType:          "k8s_secrets",
			RequiredK8sSecrets: []string{"secret-1", "secret-2", "secret-3"},
			RetryCountLimit:    10,
			RetryIntervalSec:   20,
			SanitizeEnabled:    false,
		}),
	},
	{
		description: "a valid map of envVar-based Secrets Provider settings returns a valid Config",
		settings: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			"SECRETS_DESTINATION":    "k8s_secrets",
			"K8S_SECRETS":            "secret-1,secret-2, secret-3",
			"RETRY_COUNT_LIMIT":      "10",
			"RETRY_INTERVAL_SEC":     "20",
			"REMOVE_DELETED_SECRETS": "false",
		},
		assert: assertGoodConfig(&Config{
			PodNamespace:       "test-namespace",
			StoreType:          "k8s_secrets",
			RequiredK8sSecrets: []string{"secret-1", "secret-2", "secret-3"},
			RetryCountLimit:    10,
			RetryIntervalSec:   20,
			SanitizeEnabled:    false,
		}),
	},
	{
		description: "settings configured with both annotations and envVars defer to the annotation value",
		settings: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			"SECRETS_DESTINATION":    "k8s_secrets",
			SecretsDestinationKey:    "file",
			"K8S_SECRETS":            "secret-1,secret-2,secret-3",
			RemoveDeletedSecretsKey:  "false",
			"REMOVE_DELETED_SECRETS": "true",
		},
		assert: assertGoodConfig(&Config{
			PodNamespace:       "test-namespace",
			StoreType:          "file",
			RequiredK8sSecrets: []string{},
			RetryCountLimit:    DefaultRetryCountLimit,
			RetryIntervalSec:   DefaultRetryIntervalSec,
			SanitizeEnabled:    false,
		}),
	},
	{
		description: "mixed-source config",
		settings: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			SecretsDestinationKey:    "k8s_secrets",
			"RETRY_COUNT_LIMIT":      "10",
			retryIntervalSecKey:      "20",
			"K8S_SECRETS":            "secret-1,secret-2,secret-3",
			RemoveDeletedSecretsKey:  "true",
			"REMOVE_DELETED_SECRETS": "false",
		},
		assert: assertGoodConfig(&Config{
			PodNamespace:       "test-namespace",
			StoreType:          "k8s_secrets",
			RequiredK8sSecrets: []string{"secret-1", "secret-2", "secret-3"},
			RetryCountLimit:    10,
			RetryIntervalSec:   20,
			SanitizeEnabled:    true,
		}),
	},
	{
		description: "a valid map of annotation-based settings with refresh enabled returns a valid Config",
		settings: map[string]string{
			"MY_POD_NAMESPACE":       "test-namespace",
			SecretsDestinationKey:    "file",
			SecretsRefreshEnabledKey: "true",
		},
		assert: assertGoodConfig(&Config{
			PodNamespace:           "test-namespace",
			StoreType:              "file",
			RequiredK8sSecrets:     []string{},
			RetryCountLimit:        5,
			RetryIntervalSec:       1,
			SecretsRefreshInterval: DefaultRefreshInterval,
			SanitizeEnabled:        DefaultSanitizeEnabled,
		}),
	},
}

func TestValidateAnnotations(t *testing.T) {
	for _, tc := range validateAnnotationsTestCases {
		t.Run(tc.description, func(t *testing.T) {
			errorLogs, infoLogs := ValidateAnnotations(tc.annotations)
			tc.assert(t, errorLogs, infoLogs)
		})
	}
}

func TestGatherSecretsProviderSettings(t *testing.T) {
	for _, tc := range gatherSecretsProviderSettingsTestCases {
		t.Run(tc.description, func(t *testing.T) {
			for envVar, value := range tc.env {
				os.Setenv(envVar, value)
			}

			settingsMap := GatherSecretsProviderSettings(tc.annotations)
			tc.assert(t, settingsMap)

			for envVar := range tc.env {
				os.Unsetenv(envVar)
			}
		})
	}
}

func TestValidateSecretsProviderSettings(t *testing.T) {
	for _, tc := range validateSecretsProviderSettingsTestCases {
		t.Run(tc.description, func(t *testing.T) {
			errorList, infoList := ValidateSecretsProviderSettings(tc.envAndAnnots)
			tc.assert(t, errorList, infoList)
		})
	}
}

func TestNewConfig(t *testing.T) {
	for _, tc := range newConfigTestCases {
		t.Run(tc.description, func(t *testing.T) {
			config := NewConfig(tc.settings)
			tc.assert(t, config)
		})
	}
}
