package inject

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticatorSidecarInjection(t *testing.T) {
	// Create the Admission Request (wrapped in an Admission Review) from the annotated
	// pod fixture. The goal is to use the annotations as a signal to the sidecar-injector
	// to mutate the Pod template spec.
	req, err := newTestAdmissionRequest(
		"./fixtures/authenticator-annotated-pod.json",
	)
	if !assert.NoError(t, err) {
		return
	}
	// Read the expected Pod template spec fixture
	expectedMod, err := ioutil.ReadFile(
		"./fixtures/authenticator-mutated-pod.json",
	)
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
