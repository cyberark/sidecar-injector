package pushtofile

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	validConjurPath1 = "valid/conjur/variable/path"
	validConjurPath2 = "another/valid/conjur/variable/path"
)

type secretsSpecTestCase struct {
	description string
	contents    string
	assert      func(t *testing.T, result []SecretSpec, err error)
}

func (tc secretsSpecTestCase) Run(t *testing.T) {
	t.Run(tc.description, func(t *testing.T) {
		secretsSpecs, err := NewSecretSpecs([]byte(tc.contents))
		tc.assert(t, secretsSpecs, err)
	})
}

func assertGoodSecretSpecs(expectedResult []SecretSpec) func(*testing.T, []SecretSpec, error) {
	return func(t *testing.T, result []SecretSpec, err error) {
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(
			t,
			expectedResult,
			result,
		)
	}
}

var secretsSpecTestCases = []secretsSpecTestCase{
	{
		description: "valid secret spec formats",
		contents: `
- dev/openshift/api-url
- admin-password: dev/openshift/password
`,
		assert: assertGoodSecretSpecs(
			[]SecretSpec{
				{
					Alias: "api-url",
					Path:  "dev/openshift/api-url",
				},
				{
					Alias: "admin-password",
					Path:  "dev/openshift/password",
				},
			},
		),
	},
	{
		description: "secret specs are not a list",
		contents: `
admin-password: dev/openshift/password
another-password: dev/openshift/password
`,
		assert: func(t *testing.T, result []SecretSpec, err error) {
			assert.Contains(t, err.Error(), "cannot unmarshal")
			assert.Contains(t, err.Error(), "into []pushtofile.SecretSpec")
		},
	},
	{
		description: "secret spec map with multiple keys",
		contents: `
- admin-password: dev/openshift/password 
  another-admin-password: dev/openshift/password
- dev/openshift/api-url
`,
		assert: func(t *testing.T, result []SecretSpec, err error) {
			assert.Contains(t, err.Error(), "expected a")
			assert.Contains(t, err.Error(), "on line 2")
		},
	},
	{
		description: "secret spec map value is not a string",
		contents: `
- dev/openshift/api-url
- key: 
    inner-key: inner-value
`,
		assert: func(t *testing.T, result []SecretSpec, err error) {
			assert.Contains(t, err.Error(), "expected a")
			assert.Contains(t, err.Error(), "on line 3")
		},
	},
	{
		description: "unrecognized secret spec format",
		contents: `
- dev/openshift/api-url
- api-password: dev/openshift/api-password
- - list item
`,
		assert: func(t *testing.T, result []SecretSpec, err error) {
			assert.Contains(t, err.Error(), "expected a")
			assert.Contains(t, err.Error(), "on line 4")
		},
	},
}

func TestNewSecretSpecs(t *testing.T) {
	for _, tc := range secretsSpecTestCases {
		tc.Run(t)
	}
}

func TestValidateSecretSpecPaths(t *testing.T) {
	maxLenConjurVarName := strings.Repeat("a", maxConjurVarNameLen)

	type assertFunc func(*testing.T, []error, string)

	assertNoErrors := func() assertFunc {
		return func(t *testing.T, errors []error, desc string) {
			assert.Len(t, errors, 0, desc)
		}
	}

	assertErrorsContain := func(expErrStrs ...string) assertFunc {
		return func(t *testing.T, errors []error, desc string) {
			assert.Len(t, errors, len(expErrStrs), desc)
			for i, expErrStr := range expErrStrs {
				assert.Contains(t, errors[i].Error(), expErrStr, desc)
			}
		}
	}

	testCases := []struct {
		description string
		path1       string
		path2       string
		assert      assertFunc
	}{
		{
			"valid Conjur paths",
			validConjurPath1,
			validConjurPath2,
			assertNoErrors(),
		}, {
			"null Conjur path and valid Conjur path",
			"",
			validConjurPath2,
			assertErrorsContain("null Conjur variable path"),
		}, {
			"Conjur path with trailing '/' and valid Conjur path",
			validConjurPath1 + "/",
			validConjurPath2,
			assertErrorsContain("has a trailing '/'"),
		}, {
			"Conjur path with max len var name and valid Conjur path",
			validConjurPath1 + "/" + maxLenConjurVarName,
			validConjurPath2,
			assertNoErrors(),
		}, {
			"Conjur path with oversized var name and valid Conjur path",
			validConjurPath1 + "/" + maxLenConjurVarName + "a",
			validConjurPath2,
			assertErrorsContain(fmt.Sprintf(
				"is longer than %d characters", maxConjurVarNameLen)),
		}, {
			"Two Conjur paths with trailing '/'",
			validConjurPath1 + "/",
			validConjurPath2 + "/",
			assertErrorsContain("has a trailing '/'", "has a trailing '/'"),
		},
	}

	for _, tc := range testCases {
		// Set up test case
		secretSpecs := []SecretSpec{
			{Alias: "foo", Path: tc.path1},
			{Alias: "bar", Path: tc.path2},
		}

		// Run test case
		err := validateSecretPaths(secretSpecs, "some-group-name")

		// Check result
		tc.assert(t, err, tc.description)
	}
}
