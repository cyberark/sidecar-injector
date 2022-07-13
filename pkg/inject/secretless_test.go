package inject

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretlessSidecarInjection(t *testing.T) {
	var testCases = []injectionTestCase{
		{
			description:                         "Secretless",
			annotatedPodTemplateSpecPath:        "./testdata/secretless-annotated-pod.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/secretless-mutated-pod.json",
		},
		{
			description:                         "Secretless with image name",
			annotatedPodTemplateSpecPath:        "./testdata/secretless-annotated-pod-with-image.json",
			expectedInjectedPodTemplateSpecPath: "./testdata/secretless-mutated-pod-with-image.json",
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
