package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

// Constants for Secrets Provider operation modes,
// and Defaults for some SP settings
const (
	K8s                       = "k8s_secrets"
	File                      = "file"
	ConjurMapKey              = "conjur-map"
	DefaultRetryCountLimit    = 5
	DefaultRetryIntervalSec   = 1
	MinRetryValue             = 0
	MinRefreshInterval        = time.Second
	DefaultRefreshIntervalStr = "5m"
	DefaultSanitizeEnabled    = true
)

var DefaultRefreshInterval, _ = time.ParseDuration(DefaultRefreshIntervalStr)

// Config defines the configuration parameters
// for the authentication requests
type Config struct {
	PodNamespace           string
	RequiredK8sSecrets     []string
	RetryCountLimit        int
	RetryIntervalSec       int
	StoreType              string
	SecretsRefreshInterval time.Duration
	SanitizeEnabled        bool
}

type annotationType int

// Represents each annotation input value type,
// used during input value validation
const (
	TYPESTRING annotationType = iota
	TYPEINT
	TYPEBOOL
)

type annotationRestraints struct {
	allowedType   annotationType
	allowedValues []string
}

const (
	AuthnIdentityKey      = "conjur.org/authn-identity"
	JwtTokenPath          = "conjur.org/jwt-token-path"
	ContainerModeKey      = "conjur.org/container-mode"
	SecretsDestinationKey = "conjur.org/secrets-destination"
	k8sSecretsKey         = "conjur.org/k8s-secrets"
	retryCountLimitKey    = "conjur.org/retry-count-limit"
	retryIntervalSecKey   = "conjur.org/retry-interval-sec"
	// SecretsRefreshIntervalKey is the Annotation key for setting the interval
	// for retrieving Conjur secrets and updating Kubernetes Secrets or
	// application secret files if necessary.
	SecretsRefreshIntervalKey = "conjur.org/secrets-refresh-interval"
	// SecretsRefreshEnabledKey is the Annotation key for enabling the refresh
	SecretsRefreshEnabledKey = "conjur.org/secrets-refresh-enabled"
	// RemoveDeletedSecretsKey is the annotaion key for enabling removing deleted secrets
	RemoveDeletedSecretsKey = "conjur.org/remove-deleted-secrets-enabled"
	debugLoggingKey         = "conjur.org/debug-logging"
	logTracesKey            = "conjur.org/log-traces"
	jaegerCollectorUrl      = "conjur.org/jaeger-collector-url"
)

// Define supported annotation keys for Secrets Provider config, as well as value restraints for each
var secretsProviderAnnotations = map[string]annotationRestraints{
	AuthnIdentityKey:          {TYPESTRING, []string{}},
	JwtTokenPath:              {TYPESTRING, []string{}},
	ContainerModeKey:          {TYPESTRING, []string{"init", "application", "sidecar"}},
	SecretsDestinationKey:     {TYPESTRING, []string{"file", "k8s_secrets"}},
	k8sSecretsKey:             {TYPESTRING, []string{}},
	retryCountLimitKey:        {TYPEINT, []string{}},
	retryIntervalSecKey:       {TYPEINT, []string{}},
	SecretsRefreshIntervalKey: {TYPESTRING, []string{}},
	SecretsRefreshEnabledKey:  {TYPEBOOL, []string{}},
	RemoveDeletedSecretsKey:   {TYPEBOOL, []string{}},
	debugLoggingKey:           {TYPEBOOL, []string{}},
	logTracesKey:              {TYPEBOOL, []string{}},
	jaegerCollectorUrl:        {TYPESTRING, []string{}},
}

// Define supported annotation key prefixes for Push to File config, as well as value restraints for each.
// In use, Push to File keys include a secret group ("conjur.org/conjur-secrets.{secret-group}").
// The values listed here will confirm hardcoded formatting, dynamic annotation content will
// be validated when used.
var pushToFileAnnotationPrefixes = map[string]annotationRestraints{
	"conjur.org/conjur-secrets.":             {TYPESTRING, []string{}},
	"conjur.org/conjur-secrets-policy-path.": {TYPESTRING, []string{}},
	"conjur.org/secret-file-path.":           {TYPESTRING, []string{}},
	"conjur.org/secret-file-format.":         {TYPESTRING, []string{"yaml", "json", "dotenv", "bash", "template"}},
	"conjur.org/secret-file-permissions.":    {TYPESTRING, []string{}},
	"conjur.org/secret-file-template.":       {TYPESTRING, []string{}},
}

// Define environment variables used in Secrets Provider config
var validEnvVars = []string{
	"MY_POD_NAMESPACE",
	"SECRETS_DESTINATION",
	"K8S_SECRETS",
	"RETRY_INTERVAL_SEC",
	"RETRY_COUNT_LIMIT",
	"JWT_TOKEN_PATH",
	"REMOVE_DELETED_SECRETS",
	"CONTAINER_MODE",
}

// ValidateAnnotations confirms that the provided annotations are properly
// formated, have the proper value type, and if the annotation in question
// had a defined set of accepted values, the provided value is confirmed.
// Function returns a list of Error logs, and a list of Info logs.
func ValidateAnnotations(annotations map[string]string) ([]error, []error) {
	errorList := []error{}
	infoList := []error{}

	for key, value := range annotations {
		if match, foundMap, err := validateAnnotationKey(key); err == nil {
			acceptedValueInfo := foundMap[match]
			err := validateAnnotationValue(key, value, acceptedValueInfo)
			if err != nil {
				errorList = append(errorList, err)
			}
		} else {
			infoList = append(infoList, err)
		}
	}

	return errorList, infoList
}

// GatherSecretsProviderSettings returns a string-to-string map of all provided environment
// variables and parsed, valid annotations that are concerned with Secrets Provider Config.
func GatherSecretsProviderSettings(annotations map[string]string) map[string]string {
	masterMap := make(map[string]string)

	for annotation, value := range annotations {
		if _, ok := secretsProviderAnnotations[annotation]; ok {
			masterMap[annotation] = value
		}
	}

	for _, envVar := range validEnvVars {
		value := os.Getenv(envVar)
		if value != "" {
			masterMap[envVar] = value
		}
	}

	return masterMap
}

// ValidateSecretsProviderSettings confirms that the provided environment variable and annotation
// settings yield a valid Secrets Provider configuration. Returns a list of Error logs, and a list
// of Info logs.
func ValidateSecretsProviderSettings(envAndAnnots map[string]string) ([]error, []error) {
	var errorList []error
	var infoList []error

	// PodNamespace must be configured by envVar
	if envAndAnnots["MY_POD_NAMESPACE"] == "" {
		errorList = append(errorList, fmt.Errorf(messages.CSPFK004E, "MY_POD_NAMESPACE"))
	}

	envStoreType := envAndAnnots["SECRETS_DESTINATION"]
	annotStoreType := envAndAnnots[SecretsDestinationKey]
	storeType := ""
	var err error

	switch {
	case annotStoreType == "":
		storeType, err = validateStore(envStoreType)
		if err != nil {
			errorList = append(errorList, err)
		}
	case validStoreType(annotStoreType):
		if validStoreType(envStoreType) {
			infoList = append(infoList, fmt.Errorf(messages.CSPFK012I, "StoreType", "SECRETS_DESTINATION", SecretsDestinationKey))
		}
		storeType = annotStoreType
	default:
		errorList = append(errorList, fmt.Errorf(messages.CSPFK043E, SecretsDestinationKey, annotStoreType, []string{File, K8s}))
	}
	envK8sSecretsStr := envAndAnnots["K8S_SECRETS"]
	annotK8sSecretsStr := envAndAnnots[k8sSecretsKey]
	if storeType == "k8s_secrets" {
		if envK8sSecretsStr == "" && annotK8sSecretsStr == "" {
			errorList = append(errorList, errors.New(messages.CSPFK048E))
		} else if envK8sSecretsStr != "" && annotK8sSecretsStr != "" {
			infoList = append(infoList, fmt.Errorf(messages.CSPFK012I, "RequiredK8sSecrets", "K8S_SECRETS", k8sSecretsKey))
		}
	}

	annotRetryCountLimit := envAndAnnots[retryCountLimitKey]
	envRetryCountLimit := envAndAnnots["RETRY_COUNT_LIMIT"]
	if annotRetryCountLimit != "" && envRetryCountLimit != "" {
		infoList = append(infoList, fmt.Errorf(messages.CSPFK012I, "RetryCountLimit", "RETRY_COUNT_LIMIT", retryCountLimitKey))
	}

	annotRetryIntervalSec := envAndAnnots[retryIntervalSecKey]
	envRetryIntervalSec := envAndAnnots["RETRY_INTERVAL_SEC"]
	if annotRetryIntervalSec != "" && envRetryIntervalSec != "" {
		infoList = append(infoList, fmt.Errorf(messages.CSPFK012I, "RetryIntervalSec", "RETRY_INTERVAL_SEC", retryIntervalSecKey))
	}

	annotSecretsRefreshEnable := envAndAnnots[SecretsRefreshEnabledKey]
	annotSecretsRefreshInterval := envAndAnnots[SecretsRefreshIntervalKey]
	err = validRefreshInterval(annotSecretsRefreshInterval, annotSecretsRefreshEnable, envAndAnnots)
	if err != nil {
		errorList = append(errorList, err)
	}
	return errorList, infoList
}

// NewConfig creates a new Secrets Provider configuration for a validated
// map of environment variable and annotation settings.
func NewConfig(settings map[string]string) *Config {
	podNamespace := settings["MY_POD_NAMESPACE"]

	storeType := settings[SecretsDestinationKey]
	if storeType == "" {
		storeType = settings["SECRETS_DESTINATION"]
	}

	k8sSecretsArr := []string{}
	if storeType != "file" {
		k8sSecretsStr := settings[k8sSecretsKey]
		if k8sSecretsStr != "" {
			k8sSecretsStr := strings.ReplaceAll(k8sSecretsStr, "- ", "")
			k8sSecretsArr = strings.Split(k8sSecretsStr, "\n")
			k8sSecretsArr = k8sSecretsArr[:len(k8sSecretsArr)-1]
		} else {
			k8sSecretsStr = settings["K8S_SECRETS"]
			k8sSecretsStr = strings.ReplaceAll(k8sSecretsStr, " ", "")
			k8sSecretsArr = strings.Split(k8sSecretsStr, ",")
		}
	}

	retryCountLimitStr := settings[retryCountLimitKey]
	if retryCountLimitStr == "" {
		retryCountLimitStr = settings["RETRY_COUNT_LIMIT"]
	}
	retryCountLimit := parseIntFromStringOrDefault(retryCountLimitStr, DefaultRetryCountLimit, MinRetryValue)

	retryIntervalSecStr := settings[retryIntervalSecKey]
	if retryIntervalSecStr == "" {
		retryIntervalSecStr = settings["RETRY_INTERVAL_SEC"]
	}
	retryIntervalSec := parseIntFromStringOrDefault(retryIntervalSecStr, DefaultRetryIntervalSec, MinRetryValue)

	refreshIntervalStr := settings[SecretsRefreshIntervalKey]
	refreshEnableStr := settings[SecretsRefreshEnabledKey]

	if refreshIntervalStr == "" && refreshEnableStr == "true" {
		refreshIntervalStr = DefaultRefreshIntervalStr
	}
	// ignore errors here, if the interval string is null, zero is returned
	refreshInterval, _ := time.ParseDuration(refreshIntervalStr)

	sanitizeEnableStr := settings[RemoveDeletedSecretsKey]
	if sanitizeEnableStr == "" {
		sanitizeEnableStr = settings["REMOVE_DELETED_SECRETS"]
	}
	sanitizeEnable := parseBoolFromStringOrDefault(sanitizeEnableStr, DefaultSanitizeEnabled)

	return &Config{
		PodNamespace:           podNamespace,
		RequiredK8sSecrets:     k8sSecretsArr,
		RetryCountLimit:        retryCountLimit,
		RetryIntervalSec:       retryIntervalSec,
		StoreType:              storeType,
		SecretsRefreshInterval: refreshInterval,
		SanitizeEnabled:        sanitizeEnable,
	}
}

// If the annotation being validated is for Push to File config, the ValidAnnotations function
// needs to be aware of the annotation's valid prefix in order to perform input validation,
// so this function returns:
//   - either the key, or the key's valid prefix
//   - the Map in which the key or prefix was found
//   - the success status of the operation
func validateAnnotationKey(key string) (string, map[string]annotationRestraints, error) {
	if strings.HasPrefix(key, "conjur.org/") {
		if _, ok := secretsProviderAnnotations[key]; ok {
			return key, secretsProviderAnnotations, nil
		} else if prefix, ok := valuePrefixInMapKeys(key, pushToFileAnnotationPrefixes); ok {
			return prefix, pushToFileAnnotationPrefixes, nil
		} else {
			return "", nil, fmt.Errorf(messages.CSPFK011I, key)
		}
	}
	return "", nil, nil
}

func validateAnnotationValue(key string, value string, acceptedValueInfo annotationRestraints) error {
	switch targetType := acceptedValueInfo.allowedType; targetType {
	case TYPEINT:
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf(messages.CSPFK042E, key, value, "Integer")
		}
	case TYPEBOOL:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf(messages.CSPFK042E, key, value, "Boolean")
		}
	case TYPESTRING:
		acceptedValues := acceptedValueInfo.allowedValues
		if len(acceptedValues) > 0 && !valueInArray(value, acceptedValues) {
			return fmt.Errorf(messages.CSPFK043E, key, value, acceptedValues)
		}
	}
	return nil
}

func valuePrefixInMapKeys(value string, searchMap map[string]annotationRestraints) (string, bool) {
	for key := range searchMap {
		if strings.HasPrefix(value, key) {
			return key, true
		}
	}
	return "", false
}

func valueInArray(value string, array []string) bool {
	for _, item := range array {
		if value == item {
			return true
		}
	}
	return false
}

func parseIntFromStringOrDefault(value string, defaultValue int, minValue int) int {
	valueInt, err := strconv.Atoi(value)
	if err != nil || valueInt < minValue {
		return defaultValue
	}
	return valueInt
}

func parseBoolFromStringOrDefault(value string, defaultValue bool) bool {
	valueBool, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return valueBool
}

func validStoreType(storeType string) bool {
	validStoreTypes := []string{K8s, File}
	for _, validStoreType := range validStoreTypes {
		if storeType == validStoreType {
			return true
		}
	}
	return false
}

func validateStore(envStoreType string) (string, error) {
	var err error
	storeType := ""
	switch envStoreType {
	case K8s:
		storeType = envStoreType
	case File:
		err = errors.New(messages.CSPFK047E)
	case "":
		err = errors.New(messages.CSPFK046E)
	default:
		err = fmt.Errorf(messages.CSPFK005E, "SECRETS_DESTINATION")
	}
	return storeType, err
}

func validRefreshInterval(intervalStr string, enableStr string, envAndAnnots map[string]string) error {

	var err error

	containerMode := envAndAnnots[ContainerModeKey]
	envContainerMode := envAndAnnots["CONTAINER_MODE"]
	if containerMode == "" {
		containerMode = envContainerMode
	}

	if intervalStr != "" || enableStr != "" {
		if containerMode != "sidecar" {
			return fmt.Errorf(messages.CSPFK051E, "Secrets refresh is enabled while container mode is set to", containerMode)
		}
		enabled, _ := strconv.ParseBool(enableStr)
		duration, e := time.ParseDuration(intervalStr)
		switch {
		// the user set enabled to true and did not set the interval
		case intervalStr == "" && enableStr != "":
			err = nil
		// duration can't be parsed
		case e != nil:
			err = fmt.Errorf(messages.CSPFK050E, intervalStr, e.Error())
		// check if the user explicitly set enable to false
		case !enabled && enableStr != "" && intervalStr != "":
			err = fmt.Errorf(messages.CSPFK050E, intervalStr, "Secrets refresh interval set to value while enable is false")
		// duration too small
		case duration < MinRefreshInterval:
			err = fmt.Errorf(messages.CSPFK050E, intervalStr, "Secrets refresh interval must be at least one second")
		}
	}
	return err
}
