package inject

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/authenticator/config"
	"github.com/cyberark/conjur-opentelemetry-tracer/pkg/trace"
	"github.com/cyberark/sidecar-injector/pkg/secrets/clients/conjur"
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
	NoHTTPS                       bool   // Runs an HTTP server when true
	Port                          int    // Webhook Server port
	CertFile                      string // Path to the x509 certificate for https
	KeyFile                       string // Path to the x509 private key matching `CertFile`
	SecretlessContainerImage      string // Container image for the Secretless sidecar
	AuthenticatorContainerImage   string // Container image for the K8s Authenticator sidecar
	SecretsProviderContainerImage string // Container image for the Secrets Provider sidecar
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
	SecretlessContainerImage      string // Container image for the Secretless sidecar
	AuthenticatorContainerImage   string // Container image for the K8s Authenticator sidecar
	SecretsProviderContainerImage string // Container image for the Secrets Provider
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

	requestKind := req.Kind.Kind
	log.Printf("johnodon request kind: %s", requestKind)

	if requestKind == "Secret" {
		// Parse Secret from inbound API request
		var secret corev1.Secret
		if err := json.Unmarshal(req.Object.Raw, &secret); err != nil {
			return failWithResponse(
				fmt.Sprintf("Could not unmarshal raw object: %v", err),
			)
		}
		log.Printf("johnodon dump: %v", secret)
		// Pull StringData from inbound Secret and parse for variable paths
		stringData := secret.StringData["conjur-map"]
		var keyvalues map[string]interface{}
		var secretIDs []string
		if err := yaml.Unmarshal([]byte(stringData), &keyvalues); err != nil {
			return failWithResponse(
				fmt.Sprintf("Could not unmarshal stringdata: %v", err),
			)
		}
		for k, v := range keyvalues {
			log.Printf("johnodon kv: %s = %s", k, v)
			secretIDs = append(secretIDs, v.(string))
		}
		// Setup Secrets Provider configuration
		spSettings := map[string]string{
			"CONJUR_AUTHN_LOGIN":      "host/conjur/authn-k8s/my-authenticator-id/apps/test-app-secrets-provider-p2f-injected",
			"CONJUR_ACCOUNT":          "myConjurAccount",
			"CONJUR_APPLIANCE_URL":    "https://conjur.myorg.com",
			"CONJUR_AUTHENTICATOR_ID": "my-authenticator-id",
			"CONJUR_AUTHN_URL":        "https://conjur-oss.conjur-oss.svc.cluster.local/authn-k8s/my-authenticator-id",
			"CONJUR_SSL_CERTIFICATE": `-----BEGIN CERTIFICATE-----
MIIDpDCCAoygAwIBAgIQC3ZLwRXcjA0dN5/qcgRpPjANBgkqhkiG9w0BAQsFADAY
MRYwFAYDVQQDEw1jb25qdXItb3NzLWNhMB4XDTIyMTExNzE0MzIzNVoXDTIzMTEx
NzE0MzIzNVowGzEZMBcGA1UEAxMQY29uanVyLm15b3JnLmNvbTCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBAPXuBweOXZ5FexlJ2E+/lrussV6xVhwW8Iu2
/DWxh6YN6zMvADySpmbg/Ptchqiem8QJndR/cQgZjz/NY8rhYla1IWytMWukVjqr
0RDjVZhv/+8+8FLFYSkKQKig2NW3crDgBjF7Yjzfybdq2lgYdwMYf5q4wU/owJpV
W5FvRIYCxTHh+hcaCqkvrvpdf+VmXvBMvg/y0a+2tY0Kue77voQvjf7wBkWOsVXE
EJ0BXr1GCx8KkykaMJvy8aqIrT/7VD4lKkQL9pHmofaPf+ytWkcE7eF2Teku29BF
4sugpdKD5OpghAcyOzrlWSsLA/6A2tq2kPiOVLSDIF5is8xnmU0CAwEAAaOB5jCB
4zAOBgNVHQ8BAf8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMC
MAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUtkBjyipgbrjGsNw6MiKDXd1ufBEw
gYIGA1UdEQR7MHmCEGNvbmp1ci5teW9yZy5jb22CCmNvbmp1ci1vc3OCFWNvbmp1
ci1vc3MuY29uanVyLW9zc4IZY29uanVyLW9zcy5jb25qdXItb3NzLnN2Y4InY29u
anVyLW9zcy5jb25qdXItb3NzLnN2Yy5jbHVzdGVyLmxvY2FsMA0GCSqGSIb3DQEB
CwUAA4IBAQCHmd3WmOpX3RJic0YnGdS6/tS4Y/SiIp20YnDGWZvac0yNSan+lze7
giaMGaa75cb5rxh9DleWTZVojflz6dSvzpjdMoVS5wk2wVyMgWwQUYIaP0cpdhTr
CALj3DSvHleLosWWhijTE92TAH2Nly1ItFppxzynI1+JPECh56ABsXg1OLJQkOu3
FIjbFFWcV9eZc4Hgv5Z6/zwbwTq+gat0D6NdyZ8G2BMftDqgjVJLZM2h5uXKOp7V
uD0paamBKxGtwIlLN6AXDEQZPRofanUd/95c52Wi1uR4sIAzoLqCNEnYzLW1KJoX
MLaIzQTZu5kqsRBlzl5SoUcWen3BFhrc
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDHDCCAgSgAwIBAgIRAIdastEVlBZ1lonoCxutCN4wDQYJKoZIhvcNAQELBQAw
GDEWMBQGA1UEAxMNY29uanVyLW9zcy1jYTAeFw0yMjExMTcxNDMyMzVaFw0yMzEx
MTcxNDMyMzVaMBgxFjAUBgNVBAMTDWNvbmp1ci1vc3MtY2EwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQDVwwuafmTRvG3YTcrBFXwP00QoASPhiT9M5thq
7Kmmj/A+oBJ8Io7JqGNEzUoEHmX2sJ4rqvJvn1+iBGSPNNY611Ru89RXgE4ypOug
5aip77Has8Rvw0M76EpQv8LQoCOOvHy5iTzwTICG3QGQYBdJw6swv6l/G27aGTmO
ojBNIu4ltmaS0iL8AyruMHQpkGqQuxMK5Me9ox14HH5P7+/dcGjXl4qTFFCBlUtY
BLJxGLIGhfD4se3HgHNz4asBGDQKOv1ADC/eZd5BsUSd9x0RZ1LU0aaypvHZk0B1
lRDGxdntXj9gCYO3e1RMEzFmMIcrd6p1cTuRrT0rwRamDkLhAgMBAAGjYTBfMA4G
A1UdDwEB/wQEAwICpDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDwYD
VR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUtkBjyipgbrjGsNw6MiKDXd1ufBEwDQYJ
KoZIhvcNAQELBQADggEBAKAxsY503xgjG5KEJDcR36etBNinbz8aEEOFPchTw7UJ
dE43sBqocvQa1y9CGL/uTjvI1cnB1nSlSAs1q0GUZ+qORbHKzHa4wmrog+b9nywD
draZ87qRCfrLMz95cWF+lj7CBIQu4UE75PSVNC/epD1fYQZ+Brgy2WmCNAcrbgQf
yxoOYPU2oX7Nla2idVwjQqrniujQ3O2bOJ07ur/qJjTM/5lemRofUD7Zo13W19ia
5iKLuKdR5tDrZiR+ggF0HhYipVNWrhKAQ9zl27eytuFp9NElbulhRepxJd4Ab3L5
fItt+ARWhpByCwZndQVIlK/FeuDI7BM3DpKZROCCHE0=
-----END CERTIFICATE-----
`,
		}
		customEnv := func(key string) string {
			if value, ok := spSettings[key]; ok {
				return value
			} else if value := os.Getenv(key); value != "" {
				return value
			} else {
				log.Printf("Could not retrieve setting %s from spSettings", key)
				return ""
			}
		}
		authConfig, err := config.NewConfigFromCustomEnv(ioutil.ReadFile, customEnv)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf("Could not create new authnConfig from spSettings: %v", err),
			)
		}
		retriever, err := conjur.NewSecretRetriever(authConfig)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf("Could not create new secret retriever: %v", err),
			)
		}
		// Set up tracer context
		tracerType, tracerURL := getTracerConfig()
		ctx, _, deferFunc, err := createTracer(tracerType, tracerURL)
		defer deferFunc(ctx)
		if err != nil {
			log.Printf("%v", err.Error())
		}
		// Pull secret data from Conjur
		secrets, err := retriever.Retrieve(secretIDs, ctx)
		if err != nil {
			return failWithResponse(
				fmt.Sprintf("Could not retrieve secrets: %v", err),
			)
		}
		for path, value := range secrets {
			log.Printf("johnodon %s: %s", path, string(value[:]))
		}
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
	conjurInjectVolumeStr, err := getAnnotation(
		&pod.ObjectMeta,
		annotationConjurInjectVolumesKey,
	)
	conjurInjectVolume := strings.Split(conjurInjectVolumeStr, ",")
	for i := range conjurInjectVolume {
		conjurInjectVolume[i] = strings.TrimSpace(conjurInjectVolume[i])
	}
	var sidecarConfig *PatchConfig
	annotations := make(map[string]string)
	annotations[annotationStatusKey] = "injected"

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

		imageName, err := getAnnotation(
			&pod.ObjectMeta,
			annotationContainerImageKey,
		)
		if err != nil {
			imageName = sidecarInjectorConfig.SecretlessContainerImage
		}

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
				sidecarImage:                  imageName,
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

		imageName, err := getAnnotation(
			&pod.ObjectMeta,
			annotationContainerImageKey,
		)
		if err != nil {
			imageName = sidecarInjectorConfig.AuthenticatorContainerImage
		}

		sidecarConfig = generateAuthenticatorSidecarConfig(AuthenticatorSidecarConfig{
			conjurConnConfigMapName: conjurConnConfigMapName,
			conjurAuthConfigMapName: conjurAuthConfigMapName,
			containerMode:           containerMode,
			containerName:           containerName,
			sidecarImage:            imageName,
		})

		containerVolumeMounts := ContainerVolumeMounts{}
		for _, receiveContainerName := range conjurInjectVolume {
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
	case "secrets-provider":
		containerImage, err := getAnnotation(
			&pod.ObjectMeta,
			annotationContainerImageKey,
		)
		if err != nil {
			containerImage = sidecarInjectorConfig.SecretsProviderContainerImage
			log.Printf("Using container image %s", containerImage)
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
		secretsDestination, err := getAnnotation(
			&pod.ObjectMeta,
			annotationSecretsDestinationKey,
		)
		sidecarConfig = generateSecretsProviderSidecarConfig(
			SecretsProviderSidecarConfig{
				containerMode:      containerMode,
				containerName:      containerName,
				sidecarImage:       containerImage,
				secretsDestination: secretsDestination,
			},
		)
		containerVolumeMounts := ContainerVolumeMounts{}

		for _, receiveContainerName := range conjurInjectVolume {
			containerVolumeMounts[receiveContainerName] = []corev1.VolumeMount{
				{
					Name:      "conjur-status",
					ReadOnly:  false,
					MountPath: "/conjur/status",
				},
				{
					Name:      "conjur-secrets",
					ReadOnly:  false,
					MountPath: "/conjur/secrets",
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
				SecretlessContainerImage:      whsvr.Params.SecretlessContainerImage,
				AuthenticatorContainerImage:   whsvr.Params.AuthenticatorContainerImage,
				SecretsProviderContainerImage: whsvr.Params.SecretsProviderContainerImage,
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

func getTracerConfig() (trace.TracerProviderType, string) {
	// First try to get the tracer config from annotations
	fmt.Sprintf("Getting tracer config from annotations")
	traceType, jaegerUrl, err := getTracerConfigFromAnnotations()

	// If no tracer is specified in annotations, get it from environment variables
	if err != nil || traceType == trace.NoopProviderType {
		fmt.Sprintf("Getting tracer config from environment variables")
		traceType, jaegerUrl = getTracerConfigFromEnv()
	}

	fmt.Sprintf("Tracer config: ", traceType, jaegerUrl)
	return traceType, jaegerUrl
}

func getTracerConfigFromEnv() (trace.TracerProviderType, string) {
	jaegerURL := os.Getenv("JAEGER_COLLECTOR_URL")
	if jaegerURL != "" {
		return trace.JaegerProviderType, jaegerURL
	}
	if os.Getenv("LOG_TRACES") == "true" {
		return trace.ConsoleProviderType, ""
	}
	return trace.NoopProviderType, ""
}

func getTracerConfigFromAnnotations() (trace.TracerProviderType, string, error) {
	return trace.JaegerProviderType, "", nil
}

// Create a TracerProvider, Tracer, and top-level (parent) Span
func createTracer(tracerType trace.TracerProviderType,
	tracerURL string) (context.Context, trace.Tracer, func(context.Context), error) {

	var tp trace.TracerProvider
	var span trace.Span

	// Create a background context for tracing
	ctx, cancel := context.WithCancel(context.Background())

	cleanupFunc := func(ctx context.Context) {
		if span != nil {
			span.End()
		}
		if tp != nil {
			tp.Shutdown(ctx)
		}
		cancel()
	}

	// Create a TracerProvider
	tp, err := trace.NewTracerProvider(tracerType, trace.SetGlobalProvider, trace.TracerProviderConfig{
		TracerName:        "a",
		TracerService:     "b",
		TracerEnvironment: "c",
		TracerID:          0,
		CollectorURL:      tracerURL,
		ConsoleWriter:     os.Stdout,
	})

	if err != nil {
		fmt.Sprintf(err.Error())
		return ctx, nil, cleanupFunc, err
	}

	// Create a Tracer and a top-level trace Span
	tracer := tp.Tracer("a")
	ctx, span = tracer.Start(ctx, "main")
	return ctx, tracer, cleanupFunc, err
}
