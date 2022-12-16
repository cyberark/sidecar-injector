package annotations

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockReadCloser returns an instantiation of the io.ReadCloser interface
// that does a no-op for file closing, and returns specified content
// for read operations.
func mockReadCloser(contents string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(contents))
}

func mockFileOpenerGenerator(store map[string]io.ReadCloser) fileOpener {
	return func(name string, flag int, perm os.FileMode) (io.ReadCloser, error) {
		rc, ok := store[name]
		if ok {
			return rc, nil
		}
		return nil, fmt.Errorf("file not found")
	}
}

func TestNewAnnotationsFromFile(t *testing.T) {
	// Create a mock 'fileOpener' that supports reading of a sample valid
	// annotations file.
	content := `conjur.org/conjur-secrets.test="- test-password: test/password\n"`
	mockOpener := mockFileOpenerGenerator(
		map[string]io.ReadCloser{
			"/podinfo/existent-file": mockReadCloser(content),
		})

	// Define test cases
	testCases := []struct {
		description string
		filePath    string
		expError    string
	}{
		{
			description: "Valid annotations file",
			filePath:    "/podinfo/existent-file",
			expError:    "",
		}, {
			description: "Nonexistent annotations file",
			filePath:    "/podinfo/nonexistent-file",
			expError:    "file not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := newAnnotationsFromFile(mockOpener, tc.filePath)
			if tc.expError == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tc.expError)
			}
		})
	}
}

type assertFunc func(t *testing.T, result map[string]string, err error)

type newAnnotationsFromReaderTestCase struct {
	description string
	contents    string
	assert      assertFunc
}

func assertGoodAnnotations(expected map[string]string) assertFunc {
	return func(t *testing.T, result map[string]string, err error) {
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, expected, result)
	}
}

func assertEmptyMap() assertFunc {
	return func(t *testing.T, result map[string]string, err error) {
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, map[string]string{}, result)
	}
}

func assertProperError(expectedErr string) assertFunc {
	return func(t *testing.T, result map[string]string, err error) {
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), expectedErr)
	}
}

var newAnnotationsFromReaderTestCases = []newAnnotationsFromReaderTestCase{
	{
		description: "valid file",
		contents: `conjur.org/authn-identity="host/conjur/authn-k8s/cluster/apps/inventory-api"
conjur.org/container-mode="init"
conjur.org/secrets-destination="k8s_secrets"
conjur.org/k8s-secrets="- k8s-secret-1\n- k8s-secret-2\n"
conjur.org/retry-count-limit="10"
conjur.org/retry-interval-sec="5"
conjur.org/debug-logging="true"
conjur.org/conjur-secrets.this-group="- test/url\n- test-password: test/password\n- test-username: test/username\n"
conjur.org/secret-file-path.this-group="this-relative-path"
conjur.org/secret-file-format.this-group="yaml"`,
		assert: assertGoodAnnotations(
			map[string]string{
				"conjur.org/authn-identity":                "host/conjur/authn-k8s/cluster/apps/inventory-api",
				"conjur.org/container-mode":                "init",
				"conjur.org/secrets-destination":           "k8s_secrets",
				"conjur.org/k8s-secrets":                   "- k8s-secret-1\n- k8s-secret-2\n",
				"conjur.org/retry-count-limit":             "10",
				"conjur.org/retry-interval-sec":            "5",
				"conjur.org/debug-logging":                 "true",
				"conjur.org/conjur-secrets.this-group":     "- test/url\n- test-password: test/password\n- test-username: test/username\n",
				"conjur.org/secret-file-path.this-group":   "this-relative-path",
				"conjur.org/secret-file-format.this-group": "yaml",
			},
		),
	},
	{
		description: "an empty annotations file results in an empty map",
		contents:    "",
		assert:      assertEmptyMap(),
	},
	{
		description: "malformed annotation file line with unquoted value",
		contents:    "conjur.org/container-mode=application",
		assert:      assertProperError("Annotation file line 1 is malformed"),
	},
	{
		description: "malformed annotation file line without '='",
		contents: `conjur.org/container-mode="application"
conjur.org/retry-count-limit: 5`,
		assert: assertProperError("Annotation file line 2 is malformed"),
	},
}

func TestNewAnnotationsFromReader(t *testing.T) {
	for _, tc := range newAnnotationsFromReaderTestCases {
		t.Run(tc.description, func(t *testing.T) {
			annotations, err := newAnnotationsFromReader(
				strings.NewReader(tc.contents))
			tc.assert(t, annotations, err)
		})
	}
}
