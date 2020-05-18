package inject

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"text/template"

	jsonpatch "github.com/evanphx/json-patch"
)

// applyPatchToAdmissionRequest runs an AdmissionRequest (wrapped in an AdmissionReview)
// through the sidecar-injector logic to extract a mutation patch, it then applies this
// patch to the origin Pod template spec to return the mutated Pod template spec.
func applyPatchToAdmissionRequest(reviewRequestBytes []byte) ([]byte, error) {
	req, err := NewAdmissionRequest(reviewRequestBytes)
	if err != nil {
		return nil, err
	}
	admissionRes := HandleAdmissionRequest(
		SidecarInjectorConfig{
			SecretlessContainerImage:    "secretless-image",
			AuthenticatorContainerImage: "authenticator-image",
		},
		req,
	)

	patch, err := jsonpatch.DecodePatch(admissionRes.Patch)
	if err != nil {
		return nil, err
	}

	return patch.Apply(req.Object.Raw)
}

// newTestAdmissionRequest creates an Admission Request (wrapped in a Admission Review).
// This is done by embedding a pod template spec, whose path is an argument, inside the
// shell of an example Admission Request. This method simplifies generating test
// Admission Requests.
func newTestAdmissionRequest(podTemplateSpecPath string) ([]byte, error) {
	t := template.Must(
		template.ParseFiles("./fixtures/authenticator-admission-request.tmpl.json"),
	)

	pod, err := ioutil.ReadFile(podTemplateSpecPath)
	if err != nil {
		return nil, err
	}

	var reqJSON bytes.Buffer

	err = t.Execute(&reqJSON, string(pod))
	if err != nil {
		return nil, err
	}

	var reqPrettyJSON bytes.Buffer
	err = json.Indent(&reqPrettyJSON, reqJSON.Bytes(), "", "  ")
	if err != nil {
		return nil, err
	}

	return reqPrettyJSON.Bytes(), nil
}
