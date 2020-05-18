package inject

import (
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

type SecretlessSidecarConfig struct {
	secretlessConfig              string
	conjurConnConfigMapName       string
	conjurAuthConfigMapName       string
	serviceAccountTokenVolumeName string
	sidecarImage string
}

// generateSecretlessSidecarConfig generates PatchConfig from a given secretlessConfigMapName
func generateSecretlessSidecarConfig(cfg SecretlessSidecarConfig) *PatchConfig {
	envvars := []corev1.EnvVar{
		envVarFromFieldPath("MY_POD_NAME", "metadata.name"),
		envVarFromFieldPath("MY_POD_NAMESPACE", "metadata.namespace"),
		envVarFromFieldPath("MY_POD_IP", "status.podIP"),
	}

	if crdSuffix, ok := os.LookupEnv("SECRETLESS_CRD_SUFFIX"); ok && crdSuffix != "" {
		envvars = append(
			envvars,
			envVarFromLiteral(
				"SECRETLESS_CRD_SUFFIX",
				crdSuffix,
			),
		)
	}

	if cfg.conjurConnConfigMapName != "" || cfg.conjurAuthConfigMapName != "" {
		envvars = append(envvars,
			envVarFromConfigMap("CONJUR_VERSION", cfg.conjurConnConfigMapName),
			envVarFromConfigMap("CONJUR_APPLIANCE_URL", cfg.conjurConnConfigMapName),
			envVarFromConfigMap("CONJUR_AUTHN_URL", cfg.conjurConnConfigMapName),
			envVarFromConfigMap("CONJUR_ACCOUNT", cfg.conjurConnConfigMapName),
			envVarFromConfigMap("CONJUR_SSL_CERTIFICATE", cfg.conjurConnConfigMapName),
			envVarFromConfigMap("CONJUR_AUTHN_LOGIN", cfg.conjurAuthConfigMapName))
	}

	// Allow configmgr#configspec in the SecretlessConfig annotation
	var configMgr string
	var configSpec string
	var secretlessConfigMapName string
	secretlessConfigPath := "/etc/secretless/secretless.yml"
	var volumes []corev1.Volume

	// Always add Service Account Token Volume Mount (SATVM)
	// It shouldn't be sidecar-injector's responsibility to add the SATVM, we only
	// do it here because the serviceaccount plugin which adds the SATVM is
	// executed before this plugin injects the sidecar. KEP-36 will solve this
	// ordering problem by re-running plugins after mutation. Then if we add a
	// container, the serviceaccount plugin will reprocess the manifest and add the
	// appropriate mount
	//
	// ** Remove me when KEP-36 lands **
	// also remove common.getServiceAccountTokenVolumeName
	// and calls to that in server.go
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      cfg.serviceAccountTokenVolumeName,
			ReadOnly:  true,
			MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
		},
	}

	// Three options for secretlessConfig
	// 1. configmapName
	// 2. configfile#configmapname
	// 3. k8s/crd#crdName

	// #2 Can't be passed straight through to the broker as its
	// expecting configfile#fspath

	if strings.Contains(cfg.secretlessConfig, "#") {
		// configmgr#configspec
		parts := strings.Split(cfg.secretlessConfig, "#")
		configMgr = parts[0]
		configSpec = parts[1]

		// option 2
		if configMgr == "configfile" {
			secretlessConfigMapName = configSpec
			configSpec = secretlessConfigPath
		}
	} else {
		// option 1
		// Old format, contains config map name only.
		configMgr = "configfile"
		secretlessConfigMapName = cfg.secretlessConfig
		configSpec = secretlessConfigPath
	}

	// if configMgr is k8s/crd, no further config is required.
	if configMgr == "configfile" {

		// Add configmap volume
		volumes = append(volumes, corev1.Volume{
			Name: "secretless-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretlessConfigMapName,
					},
				},
			},
		},
		)

		// add configmap mount
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "secretless-config",
			ReadOnly:  true,
			MountPath: "/etc/secretless",
		},
		)
	}

	containers := []corev1.Container{
		{
			Name:            "secretless",
			Image:           cfg.sidecarImage,
			Args:            []string{"-config-mgr", fmt.Sprintf("%s#%s", configMgr, configSpec)},
			ImagePullPolicy: "Always",
			VolumeMounts:    volumeMounts,
			Env:             envvars,
		},
	}

	return &PatchConfig{
		Containers: containers,
		Volumes:    volumes,
	}
}
