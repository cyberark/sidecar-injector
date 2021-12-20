package inject

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

type WebhookServer struct {
	Server *http.Server
	Params WebhookServerParameters
}

// Webhook Server parameters
type WebhookServerParameters struct {
	NoHTTPS                     bool   // Runs an HTTP server when true
	Port                        int    // Webhook Server port
	CertFile                    string // Path to the x509 certificate for https
	KeyFile                     string // Path to the x509 private key matching `CertFile`
	SecretlessContainerImage    string // Container image for the Secretless sidecar
	AuthenticatorContainerImage string // Container image for the Kubernetes Authenticator
	// sidecar
}

func failWithResponse(errMsg string) admissionv1.AdmissionResponse {
	log.Printf(errMsg)
	return admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Message: errMsg,
		},
	}
}

// SidecarInjectorConfig are configuration values for the sidecar injector logic
type SidecarInjectorConfig struct {
	SecretlessContainerImage    string // Container image for the Secretless sidecar
	AuthenticatorContainerImage string // Container image for the Kubernetes Authenticator
	// sidecar
}

// HandleAdmissionRequest applies the sidecar-injector logic to the AdmissionRequest
// and returns the results as an AdmissionResponse.
func HandleAdmissionRequest(
	sidecarInjectorConfig SidecarInjectorConfig,
	req *admissionv1.AdmissionRequest,
) admissionv1.AdmissionResponse {
	if req == nil {
		return failWithResponse("Received empty request")
	}

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return failWithResponse(
			fmt.Sprintf("Could not unmarshal raw object: %v", err),
		)
	}

	log.Printf(
		"AdmissionRequest for Version=%s, Kind=%s, Namespace=%v PodName=%v UID=%v rfc6902PatchOperation=%v UserInfo=%v",
		req.Kind.Version,
		req.Kind.Kind,
		req.Namespace,
		metaName(&pod.ObjectMeta),
		req.UID,
		req.Operation,
		req.UserInfo,
	)

	// Determine whether to perform mutation
	if !mutationRequired(ignoredNamespaces, &pod.ObjectMeta) {
		log.Printf(
			"Skipping mutation for %s/%s due to policy check",
			req.Namespace,
			metaName(&pod.ObjectMeta),
		)

		return admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	injectType, err := getAnnotation(&pod.ObjectMeta, annotationInjectTypeKey)
	containerMode, err := getAnnotation(&pod.ObjectMeta, annotationContainerModeKey)
	containerName, err := getAnnotation(&pod.ObjectMeta, annotationContainerNameKey)
	conjurTokenReceiversStr, err := getAnnotation(
		&pod.ObjectMeta,
		annotationConjurTokenReceiversKey,
	)
	conjurTokenReceivers := strings.Split(conjurTokenReceiversStr, ",")
	for i := range conjurTokenReceivers {
		conjurTokenReceivers[i] = strings.TrimSpace(conjurTokenReceivers[i])
	}

	var sidecarConfig *PatchConfig
	switch injectType {
	case "secretless":

		secretlessConfig, err := getAnnotation(
			&pod.ObjectMeta,
			annotationSecretlessConfigKey,
		)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf(
					"Mutation failed for pod %s, in namespace %s, due to %s",
					pod.Name,
					req.Namespace,
					err.Error(),
				),
			)
		}

		secretlessCRDSuffix, _ := getAnnotation(&pod.ObjectMeta,
			annotationSecretlessCRDSuffixKey)

		conjurConnConfigMapName, _ := getAnnotation(
			&pod.ObjectMeta,
			annotationConjurConnConfigKey,
		)
		conjurAuthConfigMapName, _ := getAnnotation(
			&pod.ObjectMeta,
			annotationConjurAuthConfigKey,
		)

		ServiceAccountTokenVolumeName, err := getServiceAccountTokenVolumeName(&pod)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf(
					"Mutation failed for pod %s, in namespace %s, due to %s",
					pod.Name,
					req.Namespace,
					err.Error(),
				),
			)
		}

		sidecarConfig = generateSecretlessSidecarConfig(
			SecretlessSidecarConfig{
				secretlessConfig:              secretlessConfig,
				secretlessCRDSuffix:           secretlessCRDSuffix,
				conjurConnConfigMapName:       conjurConnConfigMapName,
				conjurAuthConfigMapName:       conjurAuthConfigMapName,
				serviceAccountTokenVolumeName: ServiceAccountTokenVolumeName,
				sidecarImage:                  sidecarInjectorConfig.SecretlessContainerImage,
			},
		)
		break
	case "authenticator":
		conjurAuthConfigMapName, err := getAnnotation(
			&pod.ObjectMeta,
			annotationConjurAuthConfigKey,
		)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf(
					"Mutation failed for pod %s, in namespace %s, due to %s",
					pod.Name,
					req.Namespace,
					err.Error(),
				),
			)
		}

		conjurConnConfigMapName, err := getAnnotation(
			&pod.ObjectMeta,
			annotationConjurConnConfigKey,
		)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf(
					"Mutation failed for pod %s, in namespace %s, due to %s",
					pod.Name,
					req.Namespace,
					err.Error(),
				),
			)
		}

		switch containerMode {
		case "sidecar", "init", "":
			break
		default:
			return failWithResponse(
				fmt.Sprintf(
					"Mutation failed for pod %s, in namespace %s, due to %s value (%s) not supported",
					pod.Name,
					req.Namespace,
					annotationContainerModeKey,
					containerMode,
				),
			)
		}

		sidecarConfig = generateAuthenticatorSidecarConfig(AuthenticatorSidecarConfig{
			conjurConnConfigMapName: conjurConnConfigMapName,
			conjurAuthConfigMapName: conjurAuthConfigMapName,
			containerMode:           containerMode,
			containerName:           containerName,
			sidecarImage:            sidecarInjectorConfig.AuthenticatorContainerImage,
		})

		containerVolumeMounts := ContainerVolumeMounts{}
		for _, receiveContainerName := range conjurTokenReceivers {
			containerVolumeMounts[receiveContainerName] = []corev1.VolumeMount{
				{
					Name:      "conjur-access-token",
					ReadOnly:  true,
					MountPath: "/run/conjur",
				},
			}
		}
		sidecarConfig.ContainerVolumeMounts = containerVolumeMounts

		break
	default:
		errMsg := fmt.Sprintf(
			"Mutation failed for pod %s, in namespace %s, due to invalid inject type annotation value = %s",
			pod.Name,
			req.Namespace,
			injectType,
		)
		log.Printf(errMsg)

		return admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: errMsg,
			},
		}
	}

	annotations := map[string]string{annotationStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, sidecarConfig, annotations)
	if err != nil {
		return admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Printf("AdmissionResponse: patch=%v\n", printPrettyPatch(patchBytes))
	return admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook Server
func (whsvr *WebhookServer) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		log.Print("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Printf("Content-Type=%s, expecting application/json", contentType)
		http.Error(w, "invalid Content-Type, expecting `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	// Declare AdmissionResponse. This is the value that will be used to craft the
	// response on this handler.
	var admissionResponse admissionv1.AdmissionResponse

	// Decode AdmissionRequest from raw AdmissionReview bytes
	admissionRequest, err := NewAdmissionRequest(body)
	if err != nil {
		log.Printf("could not decode body: %v", err)

		// Set AdmissionResponse with error message
		admissionResponse = admissionv1.AdmissionResponse{
			UID: admissionRequest.UID,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		// Set AdmissionResponse with results from HandleAdmissionRequest
		admissionResponse = HandleAdmissionRequest(
			SidecarInjectorConfig{
				SecretlessContainerImage:    whsvr.Params.SecretlessContainerImage,
				AuthenticatorContainerImage: whsvr.Params.AuthenticatorContainerImage,
			},
			admissionRequest,
		)
	}

	// Ensure the response has the same UID as the original request (if the request field
	// was populated)
	admissionResponse.UID = admissionRequest.UID

	// Wrap AdmissonResponse in AdmissionReview, then marshal it to JSON
	resp, err := json.Marshal(admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &admissionResponse,
	})
	if err != nil {
		log.Printf("could not encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	log.Printf("Ready to write response ...")
	if _, err := w.Write(resp); err != nil {
		log.Printf("could not write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// NewAdmissionRequest parses raw bytes to create an AdmissionRequest. AdmissionRequest
// actually comes wrapped inside the bytes of an AdmissionReview.
func NewAdmissionRequest(reviewRequestBytes []byte) (*admissionv1.AdmissionRequest, error) {
	var ar admissionv1.AdmissionReview
	_, _, err := deserializer.Decode(reviewRequestBytes, nil, &ar)

	log.Printf("Received AdmissionReview, APIVersion: %s, Kind: %s\n", ar.APIVersion, ar.Kind)
	return ar.Request, err
}
