package pushtofile

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

type pushToWriterTestCase struct {
	description string
	template    string
	secrets     []*Secret
	assert      func(*testing.T, string, error)
}

func (tc pushToWriterTestCase) Run(t *testing.T) {
	t.Run(tc.description, func(t *testing.T) {
		buf := new(bytes.Buffer)
		_, err := pushToWriter(
			buf,
			"group path",
			tc.template,
			tc.secrets,
		)
		tc.assert(t, buf.String(), err)
	})
}

func assertGoodOutput(expected string) func(*testing.T, string, error) {
	return func(t *testing.T, actual string, err error) {
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(
			t,
			expected,
			actual,
		)
	}
}

var writeToFileTestCases = []pushToWriterTestCase{
	{
		description: "happy path",
		template:    `{{secret "alias"}}`,
		secrets:     []*Secret{{Alias: "alias", Value: "secret value"}},
		assert:      assertGoodOutput("secret value"),
	},
	{
		description: "undefined secret",
		template:    `{{secret "x"}}`,
		secrets:     []*Secret{{Alias: "some alias", Value: "secret value"}},
		assert: func(t *testing.T, s string, err error) {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), `secret alias "x" not present in specified secrets for group`)
		},
	},
	{
		// Conversions defined in Go source:
		// https://cs.opensource.google/go/go/+/refs/tags/go1.17.2:src/text/template/funcs.go;l=608
		description: "confirm use of built-in html escape template function",
		template:    `{{secret "alias" | html}}`,
		secrets:     []*Secret{{Alias: "alias", Value: "\" ' & < > \000"}},
		assert:      assertGoodOutput("&#34; &#39; &amp; &lt; &gt; \uFFFD"),
	},
	{
		description: "base64 encoding",
		template:    `{{secret "alias" | b64enc}}`,
		secrets:     []*Secret{{Alias: "alias", Value: "secret value"}},
		assert:      assertGoodOutput("c2VjcmV0IHZhbHVl"),
	},
	{
		description: "base64 decoding",
		template:    `{{secret "alias" | b64dec}}`,
		secrets:     []*Secret{{Alias: "alias", Value: "c2VjcmV0IHZhbHVl"}},
		assert:      assertGoodOutput("secret value"),
	},
	{
		description: "base64 decoding invalid input",
		template:    `{{secret "alias" | b64dec}}`,
		secrets:     []*Secret{{Alias: "alias", Value: "c2VjcmV0IHZhbHVl_invalid"}},
		assert: func(t *testing.T, s string, err error) {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "value could not be base64 decoded")
			// Ensure the error doesn't contain the actual secret
			assert.NotContains(t, err.Error(), "c2VjcmV0IHZhbHVl_invalid")
		},
	},
	{
		description: "iterate over secret key-value pairs",
		template: `{{- range $index, $secret := .SecretsArray -}}
{{- if $index }}
{{ end }}
{{- $secret.Alias }}: {{ $secret.Value }}
{{- end -}}`,
		secrets: []*Secret{
			{Alias: "environment", Value: "prod"},
			{Alias: "url", Value: "https://example.com"},
			{Alias: "username", Value: "example-user"},
			{Alias: "password", Value: "example-pass"},
		},
		assert: assertGoodOutput(`environment: prod
url: https://example.com
username: example-user
password: example-pass`),
	},
	{
		description: "nested templates",
		template: `{{- define "contents" -}}
Alias : {{ .Alias }}
Value : {{ .Value }}
{{ end }}
{{- define "parent" -}}
Nested Template
{{ template "contents" . -}}
===============
{{ end }}
{{- range $index, $secret := .SecretsArray -}}
{{ template "parent" . }}
{{- end -}}`,
		secrets: []*Secret{
			{Alias: "environment", Value: "prod"},
			{Alias: "url", Value: "https://example.com"},
			{Alias: "username", Value: "example-user"},
			{Alias: "password", Value: "example-pass"},
		},
		assert: assertGoodOutput(`Nested Template
Alias : environment
Value : prod
===============
Nested Template
Alias : url
Value : https://example.com
===============
Nested Template
Alias : username
Value : example-user
===============
Nested Template
Alias : password
Value : example-pass
===============
`),
	},
}

func Test_pushToWriter(t *testing.T) {
	for _, tc := range writeToFileTestCases {
		tc.Run(t)
	}
}

func Test_pushToWriter_contentChanges(t *testing.T) {
	t.Run("content changes", func(t *testing.T) {
		// Call pushToWriter with a simple template and secret.
		secrets := []*Secret{{Alias: "alias", Value: "secret value"}}
		groupName := "group path"
		template := `{{secret "alias"}}`

		buf := new(bytes.Buffer)
		updated, err := pushToWriter(
			buf,
			groupName,
			template,
			secrets,
		)
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, "secret value", buf.String())

		// Now clear the buffer and call pushToWriter again. Since the secret is the same,
		// it should not update the buffer.
		buf.Reset()
		updated, err = pushToWriter(
			buf,
			groupName,
			template,
			secrets,
		)

		assert.NoError(t, err)
		assert.False(t, updated)
		assert.Zero(t, buf.Len())

		// Now change the secret and call pushToWriter again. This time, the buffer should
		// be updated because the secret has changed.
		updated, err = pushToWriter(
			buf,
			groupName,
			template,
			[]*Secret{{Alias: "alias", Value: "secret changed"}},
		)
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, "secret changed", buf.String())

		// Repeat the test but this time change the template instead of the secret. The buffer should still
		// be updated because the rendered output should be different.
		buf.Reset()
		updated, err = pushToWriter(
			buf,
			groupName,
			`- {{secret "alias"}}`,
			[]*Secret{{Alias: "alias", Value: "secret changed"}},
		)
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, "- secret changed", buf.String())
	})
}
