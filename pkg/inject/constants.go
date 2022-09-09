package inject

const (
	annotationConjurAuthConfigKey     = "conjur.org/conjurAuthConfig"
	annotationConjurConnConfigKey     = "conjur.org/conjurConnConfig"
	annotationContainerNameKey        = "conjur.org/container-name"
	annotationContainerModeKey        = "conjur.org/container-mode"
	annotationConjurInjectVolumesKey = "conjur.org/conjur-inject-volumes"
	annotationInjectKey               = "conjur.org/inject"
	annotationInjectTypeKey           = "conjur.org/inject-type"
	annotationSecretlessConfigKey     = "conjur.org/secretless-config"
	annotationSecretlessCRDSuffixKey  = "conjur.org/secretless-CRD-suffix"
	annotationStatusKey               = "conjur.org/status"
	annotationContainerImageKey       = "conjur.org/container-image"
	annotationSecretsDestinationKey   = "conjur.org/secrets-destination"
)
// These annotations are only used for sidecar injector and not passed on to the
// injected container
var sidecarInjectorAnnot = []string {
	annotationConjurAuthConfigKey,
	annotationConjurConnConfigKey,
	annotationContainerNameKey,
	annotationConjurInjectVolumesKey,
	annotationInjectKey,
	annotationInjectTypeKey,
	annotationSecretlessConfigKey,
	annotationSecretlessCRDSuffixKey,
	annotationContainerImageKey,
}