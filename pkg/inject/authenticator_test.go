package inject

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"
	"text/template"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticatorSidecarInjection(t *testing.T) {
	// Create the Admission Request (wrapped in an Admission Review) from the annotated
	// pod fixture. The goal is to use the annotations as a signal to the sidecar-injector
	// to mutate the Pod template spec.
	req, err := newTestAdmissionRequest("./fixtures/authenticator-annotated-pod.json")
	if !assert.NoError(t, err) {
		return
	}
	// Read the expected Pod template spec fixture
	expectedMod, err := ioutil.ReadFile("./fixtures/authenticator-mutated-pod.json")
	if !assert.NoError(t, err) {
		return
	}

	// Get the modified Pod template spec from the input .
	mod, err := applyPatchToAdmissionRequest(req)
	if !assert.NoError(t, err) {
		return
	}

	// Assert that the modified Pod template spec should equal the expected Pod template spec
	assert.JSONEq(t, string(expectedMod), string(mod))
}

// applyPatchToAdmissionRequest runs an AdmissionRequest (wrapped in an AdmissionReview)
// through the sidecar-injector logic to extract a mutation patch, it then applies this
// patch to the origin Pod template spec to return the mutated Pod template spec.
func applyPatchToAdmissionRequest(reviewRequestBytes []byte) ([]byte, error) {
	req, err := NewAdmissionRequest(reviewRequestBytes)
	if err != nil {
		return nil, err
	}
	admissionRes := HandleAdmissionRequest(req)

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
