package inject

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

type injectionTestCase struct {
	description                         string
	annotatedPodTemplateSpecPath        string
	expectedInjectedPodTemplateSpecPath string
}

func TestSidecarInjection(t *testing.T) {
	var testCases = []injectionTestCase{
		{
			description:                         "Secretless",
			annotatedPodTemplateSpecPath:        "./testdata/secretless-annotated-pod.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/secretless-mutated-pod.json",
		},
		{
			description:                         "Kubernetes Authenticator",
			annotatedPodTemplateSpecPath:        "./testdata/authenticator-annotated-pod.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/authenticator-mutated-pod.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create the Admission Request (wrapped in an Admission Review) from the
			// annotated pod fixture. The goal is to use the annotations as a signal to
			// the sidecar-injector to mutate the Pod template spec.
			req, err := newTestAdmissionRequest(
				tc.annotatedPodTemplateSpecPath,
			)
			if !assert.NoError(t, err) {
				return
			}

			// Read the expected Pod template spec fixture.
			expectedMod, err := ioutil.ReadFile(
				tc.expectedInjectedPodTemplateSpecPath,
			)
			if !assert.NoError(t, err) {
				return
			}

			// Get the modified Pod template spec from the input.
			mod, err := applyPatchToAdmissionRequest(req)
			if !assert.NoError(t, err) {
				return
			}

			// Assert that the modified Pod template spec should equal the expected Pod
			// template spec.
			assert.JSONEq(t, string(expectedMod), string(mod))
		})
	}
}
