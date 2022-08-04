package inject

import (
	corev1 "k8s.io/api/core/v1"
	"os"
)

type SecretsProviderSidecarConfig struct {
	containerMode           string
	containerName           string
	sidecarImage            string
	secretsDestination      string
}
var validEnvVars = []string{
	"CONJUR_ACCOUNT",
	"CONJUR_APPLIANCE_URL",
	"CONJUR_AUTHENTICATOR_ID",
	"CONJUR_AUTHN_URL",
	"CONJUR_SSL_CERTIFICATE",
}

// generateSecretsProviderSidecarConfig generates PatchConfig from a
// given SecretsProviderSidecarConfig
func generateSecretsProviderSidecarConfig(
	cfg SecretsProviderSidecarConfig,
    ) *PatchConfig {
	var containers, initContainers []corev1.Container
	masterMap := make(map[string]string)
	for _, envVar := range validEnvVars {
		value := os.Getenv(envVar)
		if value != "" {
			masterMap[envVar] = value
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
		volumeMounts = append(volumeMounts,volumeMount)
	}
	container := corev1.Container{
		Name:            cfg.containerName,
		Image:           cfg.sidecarImage,
		ImagePullPolicy: "Always",
		VolumeMounts: volumeMounts,
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
				masterMap["CONJUR_ACCOUNT"],
			),
			envVarFromLiteral(
				"CONJUR_APPLIANCE_URL",
				masterMap["CONJUR_APPLIANCE_URL"],
			),
			envVarFromLiteral(
				"CONJUR_AUTHENTICATOR_ID",
				masterMap["CONJUR_AUTHENTICATOR_ID"],
			),
			envVarFromLiteral(
				"CONJUR_AUTHN_URL",
				masterMap["CONJUR_AUTHN_URL"],
			),
			envVarFromLiteral(
				"CONJUR_SSL_CERTIFICATE",
				masterMap["CONJUR_SSL_CERTIFICATE"],
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
		Volumes: volumes,
	}
}

func getSPVolumes( secretsDest string ) []corev1.Volume {

	//var vol []corev1.Volume
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

