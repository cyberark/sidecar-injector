package inject

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mutationRequired determines if target resource requires mutation
func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	// skip special kubernete system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip mutation for %v for it' in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}

	status, _ := getAnnotation(metadata, annotationStatusKey)

	// determine whether to perform mutation based on annotation for the target resource
	required := strings.ToLower(status) != "injected"
	if required {
		switch injectValue, _ := getAnnotation(metadata, annotationInjectKey); strings.ToLower(injectValue) {
		case "y", "yes", "true", "on":
			required = true
		default:
			required = false
		}
	}

	glog.Infof("Mutation policy for %v/%v: status: %q required:%v", metadata.Namespace, metadata.Name, status, required)
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
	return corev1.EnvVar {
		Name: envVarName,
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
	return "", errors.New("Service account token volume mount not found")
}
