package inject

import (
	corev1 "k8s.io/api/core/v1"
	"os"
)

type SecretsProviderSidecarConfig struct {
	containerMode      string
	containerName      string
	sidecarImage       string
	secretsDestination string
}

var conjurEnvVars = []string{
	"CONJUR_ACCOUNT",
	"conjurAccount",
	"CONJUR_APPLIANCE_URL",
	"conjurApplianceUrl",
	"CONJUR_AUTHENTICATOR_ID",
	"authnK8sAuthenticatorID",
	"CONJUR_AUTHN_URL",
	"CONJUR_SSL_CERTIFICATE",
	"conjurSslCertificate",
}

func getConjurAuthnURL(envVars map[string]string) string {

	authnMethod := "authn-k8s"
	// If "CONJUR_AUTHN_URL" is explicitly set, use it
	if envVars["CONJUR_AUTHN_URL"] != "" {
		return envVars["CONJUR_AUTHN_URL"]
	}
	return envVars["conjurApplianceUrl"] + "/" + authnMethod + "/" +
		envVars["authnK8sAuthenticatorID"]
}
func getConjurEnv(envVars map[string]string, primary string,
	secondary string) string {

	if envVars[primary] != "" {
		return envVars[primary]
	}
	return envVars[secondary]
}

// generateSecretsProviderSidecarConfig generates PatchConfig from a
// given SecretsProviderSidecarConfig
func generateSecretsProviderSidecarConfig(
	cfg SecretsProviderSidecarConfig,
) *PatchConfig {
	var containers, initContainers []corev1.Container
	envVars := make(map[string]string)
	for _, envVar := range conjurEnvVars {
		value := os.Getenv(envVar)
		if value != "" {
			envVars[envVar] = value
		}
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "podinfo",
			ReadOnly:  true,
			MountPath: "/conjur/podinfo",
		},
		{
			Name:      "conjur-status",
			ReadOnly:  false,
			MountPath: "/conjur/status",
		},
	}
	if cfg.secretsDestination == "file" {
		volumeMount := corev1.VolumeMount{
			Name:      "conjur-secrets",
			ReadOnly:  false,
			MountPath: "/conjur/secrets",
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}
	container := corev1.Container{
		Name:            cfg.containerName,
		Image:           cfg.sidecarImage,
		ImagePullPolicy: "Always",
		VolumeMounts:    volumeMounts,
		Env: []corev1.EnvVar{
			envVarFromFieldPath(
				"MY_POD_NAME",
				"metadata.name",
			),
			envVarFromFieldPath(
				"MY_POD_NAMESPACE",
				"metadata.namespace",
			),
			envVarFromLiteral(
				"CONJUR_ACCOUNT",
				getConjurEnv(envVars, "CONJUR_ACCOUNT", "conjurAccount"),
			),
			envVarFromLiteral(
				"CONJUR_APPLIANCE_URL",
				getConjurEnv(envVars, "CONJUR_APPLIANCE_URL", "conjurApplianceUrl"),
			),
			envVarFromLiteral(
				"CONJUR_AUTHENTICATOR_ID",
				getConjurEnv(envVars, "CONJUR_AUTHENTICATOR_ID", "authnK8sAuthenticatorID"),
			),
			envVarFromLiteral(
				"CONJUR_AUTHN_URL",
				getConjurAuthnURL(envVars),
			),
			envVarFromLiteral(
				"CONJUR_SSL_CERTIFICATE",
				getConjurEnv(envVars, "CONJUR_SSL_CERTIFICATE", "conjurSslCertificate"),
			),
		},
	}

	candidates := []corev1.Container{container}
	if cfg.containerMode == "init" {
		initContainers = candidates
	} else {
		containers = candidates
	}
	volumes := getSPVolumes(cfg.secretsDestination)

	return &PatchConfig{
		Containers:     containers,
		InitContainers: initContainers,
		Volumes:        volumes,
	}
}

func getSPVolumes(secretsDest string) []corev1.Volume {

	var volume corev1.Volume

	volumes := []corev1.Volume{
		{
			Name: "podinfo",
			VolumeSource: corev1.VolumeSource{
				DownwardAPI: &corev1.DownwardAPIVolumeSource{
					Items: []corev1.DownwardAPIVolumeFile{
						{
							Path: "annotations",
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.annotations",
							},
						},
					},
				},
			},
		},
		{
			Name: "conjur-status",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: "Memory",
				},
			},
		},
	}
	if secretsDest == "file" {
		volume = corev1.Volume{
			Name: "conjur-secrets",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: "Memory",
				},
			},
		}
		volumes = append(volumes, volume)
	}
	return volumes
}
