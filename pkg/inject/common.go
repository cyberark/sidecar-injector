package inject

import (
	"errors"
	"fmt"
	"log"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func metaName(meta *metav1.ObjectMeta) string {
	name := meta.GenerateName
	if name == "" {
		name = meta.Name
	}

	return name
}

// mutationRequired determines if target resource requires mutation
func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	// skip special Kubernetes system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			log.Printf(
				"Skip mutation for %v for it' in special namespace:%v",
				metaName(metadata),
				metadata.Namespace,
			)
			return false
		}
	}

	injectedStatus, _ := getAnnotation(metadata, annotationStatusKey)

	// determine whether to perform mutation based on annotation for the target resource
	required := strings.ToLower(injectedStatus) != "injected"
	if required {
		injectValue, _ := getAnnotation(metadata, annotationInjectKey)
		switch strings.ToLower(injectValue) {
		case "y", "yes", "true", "on":
			required = true
		default:
			required = false
		}
	}

	log.Printf(
		"Mutation policy for %s/%s: injected status: %q required:%v",
		metaName(metadata),
		metadata.Name,
		injectedStatus,
		required,
	)

	return required
}

func envVarFromConfigMap(envVarName, configMapName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{

			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
				Key: envVarName,
			},
		},
	}
}

func envVarFromFieldPath(envVarName, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

func envVarFromLiteral(envVarName, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  envVarName,
		Value: value,
	}
}

func getAnnotation(metadata *metav1.ObjectMeta, key string) (string, error) {
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	value, hasKey := annotations[key]
	if !hasKey {
		return "", fmt.Errorf("missing annotation %s", key)
	}
	return value, nil
}

/**
 * Given a pod, return the name of the volume that contains
 * the service account token. VolumeMounts are used instead
 * of volumes so that the mount path can be matched on.
 * Otherwise a name pattern match would be required and
 * that could have unexpected results.
 */
func getServiceAccountTokenVolumeName(pod *corev1.Pod) (string, error) {
	for _, container := range pod.Spec.Containers {
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.MountPath == "/var/run/secrets/kubernetes.io/serviceaccount" {
				return volumeMount.Name, nil
			}
		}
	}

	return "", errors.New("service account token volume mount not found")
}
