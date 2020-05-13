package inject

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// RFC6902 JSON patches
type rfc6902PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// RFC6902 JSON patch operations
const (
	patchOperationAdd     = "add"
	patchOperationReplace = "replace"
)

// create mutation patch for resources
func createPatch(pod *corev1.Pod, sidecarConfig *PatchConfig, annotations map[string]string) ([]byte, error) {
	var patch []rfc6902PatchOperation

	patch = append(patch, addContainer(pod.Spec.InitContainers, sidecarConfig.InitContainers, "/spec/initContainers")...)
	patch = append(patch, addContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
	patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)
	patch = append(
		patch,
		addVolumeMounts(
			pod.Spec.Containers,
			sidecarConfig.ContainerVolumeMounts,
			"/spec/containers",
		)...,
	)

	return json.Marshal(patch)
}

// addContainer create a patch for adding containers
func addContainer(target, added []corev1.Container, basePath string) (patch []rfc6902PatchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, rfc6902PatchOperation{
			Op:    patchOperationAdd,
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// addVolumeMounts creates a patch for adding volume mounts
func addVolumeMounts(target []corev1.Container, added ContainerVolumeMounts, basePath string) (patch []rfc6902PatchOperation) {
	for index, container := range target {
		volumeMounts, ok := added[container.Name]
		if !ok || len(volumeMounts) == 0 { continue }

		if len(container.VolumeMounts) == 0 {
			volumeMount := volumeMounts[0]
			volumeMounts = volumeMounts[1:]

			path := fmt.Sprintf("%s/%d/volumeMounts", basePath, index)
			patch = append(patch, rfc6902PatchOperation{
				Op:    patchOperationAdd,
				Path:  path,
				Value: []corev1.VolumeMount{volumeMount},
			})
		}

		path := fmt.Sprintf("%s/%d/volumeMounts/-", basePath, index)
		for _, volumeMount := range volumeMounts {
			patch = append(patch, rfc6902PatchOperation{
				Op:    patchOperationAdd,
				Path:  path,
				Value: volumeMount,
			})
		}
	}
	return patch
}

// addVolume creates a patch for adding volumes
func addVolume(target, added []corev1.Volume, basePath string) (patch []rfc6902PatchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, rfc6902PatchOperation{
			Op:    patchOperationAdd,
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// updateAnnotation creates a patch for adding/updating annotations
func updateAnnotation(target, added map[string]string) (patch []rfc6902PatchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, rfc6902PatchOperation{
				Op:   patchOperationAdd,
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, rfc6902PatchOperation{
				Op:    patchOperationReplace,
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}
