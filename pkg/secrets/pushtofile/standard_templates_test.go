package pushtofile

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	invalidYAMLChar    = "invalid YAML character"
	invalidJSONChar    = "invalid JSON character"
	yamlAliasTooLong   = "too long for YAML"
	jsonAliasTooLong   = "too long for JSON"
	invalidBashVarName = "can only include alphanumerics and underscores"
	validConjurPath    = "valid/conjur/variable/path"
)

type assertErrorFunc func(*testing.T, error, string)

func assertNoError() assertErrorFunc {
	return func(t *testing.T, err error, desc string) {
		assert.NoError(t, err, desc)
	}
}

func assertErrorContains(expErrStr string) assertErrorFunc {
	return func(t *testing.T, err error, desc string) {
		assert.Error(t, err, desc)
		assert.Contains(t, err.Error(), expErrStr, desc)
	}
}

var standardTemplateTestCases = []pushToWriterTestCase{
	{
		description: "json",
		template:    standardTemplates["json"].template,
		secrets: []*Secret{
			{Alias: "alias 1", Value: "secret value 1"},
			{"alias 2", "secret value 2"},
		},
		assert: assertGoodOutput(`{"alias 1":"secret value 1","alias 2":"secret value 2"}`),
	},
	{
		description: "yaml",
		template:    standardTemplates["yaml"].template,
		secrets: []*Secret{
			{Alias: "alias 1", Value: "secret value 1"},
			{"alias 2", "secret value 2"},
		},
		assert: assertGoodOutput(`"alias 1": "secret value 1"
"alias 2": "secret value 2"`),
	},
	{
		description: "dotenv",
		template:    standardTemplates["dotenv"].template,
		secrets: []*Secret{
			{Alias: "alias1", Value: "secret value 1"},
			{"alias2", "secret value 2"},
		},
		assert: assertGoodOutput(`alias1="secret value 1"
alias2="secret value 2"`),
	},
	{
		description: "bash",
		template:    standardTemplates["bash"].template,
		secrets: []*Secret{
			{Alias: "alias1", Value: "secret value 1"},
			{"alias2", "secret value 2"},
		},
		assert: assertGoodOutput(`export alias1="secret value 1"
export alias2="secret value 2"`),
	},
}

func Test_standardTemplates(t *testing.T) {
	for _, tc := range standardTemplateTestCases {
		tc.Run(t)
	}
}

type aliasCharTestCase struct {
	description string
	testChar    rune
	assert      assertErrorFunc
}

func (tc *aliasCharTestCase) Run(t *testing.T, fileFormat string) {
	t.Run(tc.description, func(t *testing.T) {
		// Set up test case
		desc := fmt.Sprintf("%s file format, key containing %s character",
			fileFormat, tc.description)
		alias := "key_containing_" + string(tc.testChar) + "_character"
		secretSpecs := []SecretSpec{{Alias: alias, Path: validConjurPath}}

		// Run test case
		_, err := FileTemplateForFormat(fileFormat, secretSpecs)

		// Check result
		tc.assert(t, err, desc)
	})
}

type aliasLenTestCase struct {
	description string
	alias       string
	assert      assertErrorFunc
}

func (tc *aliasLenTestCase) Run(t *testing.T, fileFormat string) {
	t.Run(tc.description, func(t *testing.T) {
		// Set up test case
		desc := fmt.Sprintf("%s file format, %s", fileFormat, tc.description)
		secretSpecs := []SecretSpec{{Alias: tc.alias, Path: validConjurPath}}

		// Run test case
		_, err := FileTemplateForFormat(fileFormat, secretSpecs)

		// Check result
		tc.assert(t, err, desc)
	})
}

func TestValidateAliasForYAML(t *testing.T) {
	testValidateAliasCharForYAML(t)
	testValidateAliasLenForYAML(t)
}

func testValidateAliasCharForYAML(t *testing.T) {
	testCases := []aliasCharTestCase{
		// YAML file format, 8-bit characters
		{"printable ASCII", '\u003F', assertNoError()},
		{"heart emoji", '💙', assertNoError()},
		{"dog emoji", '🐶', assertNoError()},
		{"ASCII NULL", '\u0000', assertErrorContains(invalidYAMLChar)},
		{"ASCII BS", '\u0008', assertErrorContains(invalidYAMLChar)},
		{"ASCII tab", '\u0009', assertNoError()},
		{"ASCII LF", '\u000A', assertNoError()},
		{"ASCII VT", '\u000B', assertErrorContains(invalidYAMLChar)},
		{"ASCII CR", '\u000D', assertNoError()},
		{"ASCII space", '\u0020', assertNoError()},
		{"ASCII tilde", '\u007E', assertNoError()},
		{"ASCII DEL", '\u007F', assertErrorContains(invalidYAMLChar)},
		// YAML file format, 16-bit Unicode
		{"Unicode NEL", '\u0085', assertNoError()},
		{"Unicode 0x86", '\u0086', assertErrorContains(invalidYAMLChar)},
		{"Unicode 0x9F", '\u009F', assertErrorContains(invalidYAMLChar)},
		{"Unicode 0xA0", '\u00A0', assertNoError()},
		{"Unicode 0xD7FF", '\uD7FF', assertNoError()},
		{"Unicode 0xE000", '\uE000', assertNoError()},
		{"Unicode 0xFFFD", '\uFFFD', assertNoError()},
		{"Unicode 0xFFFE", '\uFFFE', assertErrorContains(invalidYAMLChar)},
		// YAML file format, 32-bit Unicode
		{"Unicode 0x10000", '\U00010000', assertNoError()},
		{"Unicode 0x10FFFF", '\U0010FFFF', assertNoError()},
	}

	for _, tc := range testCases {
		tc.Run(t, "yaml")
	}
}

func testValidateAliasLenForYAML(t *testing.T) {
	maxLenAlias := strings.Repeat("a", maxYAMLKeyLen)

	testCases := []aliasLenTestCase{
		{"single char alias", "a", assertNoError()},
		{"maximum length alias", maxLenAlias, assertNoError()},
		{"oversized alias", maxLenAlias + "a", assertErrorContains(yamlAliasTooLong)},
	}

	for _, tc := range testCases {
		tc.Run(t, "yaml")
	}
}

func TestValidateAliasForJSON(t *testing.T) {
	testValidateAliasCharForJSON(t)
	testValidateAliasLenForJSON(t)
}

func testValidateAliasCharForJSON(t *testing.T) {
	testCases := []aliasCharTestCase{
		// JSON file format, valid characters
		{"ASCII space", '\u0020', assertNoError()},
		{"ASCII tilde", '~', assertNoError()},
		{"heart emoji", '💙', assertNoError()},
		{"dog emoji", '🐶', assertNoError()},
		{"Unicode 0x10000", '\U00010000', assertNoError()},
		{"Unicode 0x10FFFF", '\U0010FFFF', assertNoError()},
		// JSON file format, invalid characters
		{"ASCII NUL", '\u0000', assertErrorContains(invalidJSONChar)},
		{"ASCII 0x1F", '\u001F', assertErrorContains(invalidJSONChar)},
		{"ASCII NULL", '\u0000', assertErrorContains(invalidJSONChar)},
		{"ASCII BS", '\u0008', assertErrorContains(invalidJSONChar)},
		{"ASCII tab", '\u0009', assertErrorContains(invalidJSONChar)},
		{"ASCII LF", '\u000A', assertErrorContains(invalidJSONChar)},
		{"ASCII VT", '\u000B', assertErrorContains(invalidJSONChar)},
		{"ASCII DEL", '\u007F', assertErrorContains(invalidJSONChar)},
		{"ASCII quote", '"', assertErrorContains(invalidJSONChar)},
		{"ASCII backslash", '\\', assertErrorContains(invalidJSONChar)},
	}

	for _, tc := range testCases {
		tc.Run(t, "json")
	}
}

func testValidateAliasLenForJSON(t *testing.T) {
	maxLenAlias := strings.Repeat("a", maxJSONKeyLen)

	testCases := []aliasLenTestCase{
		{"single-char alias", "a", assertNoError()},
		{"maximum length alias", maxLenAlias, assertNoError()},
		{"oversized alias", maxLenAlias + "a", assertErrorContains(jsonAliasTooLong)},
	}

	for _, tc := range testCases {
		tc.Run(t, "json")
	}
}

func TestValidateAliasForBash(t *testing.T) {
	testValidateAliasForBashOrDotenv(t, "bash")
}

func TestValidateAliasForDotenv(t *testing.T) {
	testValidateAliasForBashOrDotenv(t, "dotenv")
}

func testValidateAliasForBashOrDotenv(t *testing.T, fileFormat string) {
	testCases := []struct {
		description string
		alias       string
		assert      assertErrorFunc
	}{
		// Bash file format, valid aliases
		{"all lower case chars", "foobar", assertNoError()},
		{"all upper case chars", "FOOBAR", assertNoError()},
		{"upper case, lower case, and underscores", "_Foo_Bar_", assertNoError()},
		{"leading underscore with digits", "_12345", assertNoError()},
		{"upper case, lower case, underscores, digits", "_Foo_Bar_1234", assertNoError()},

		// Bash file format, invalid aliases
		{"leading digit", "7th_Heaven", assertErrorContains(invalidBashVarName)},
		{"spaces", "FOO BAR", assertErrorContains(invalidBashVarName)},
		{"dashes", "FOO-BAR", assertErrorContains(invalidBashVarName)},
		{"single quotes", "FOO_'BAR'", assertErrorContains(invalidBashVarName)},
		{"dog emoji", "FOO_'🐶'_BAR", assertErrorContains(invalidBashVarName)},
		{"trailing space", "FOO_BAR ", assertErrorContains(invalidBashVarName)},
	}

	for _, tc := range testCases {
		// Set up test case
		desc := fmt.Sprintf("%s file format, alias with %s",
			fileFormat, tc.description)
		secretSpecs := []SecretSpec{{Alias: tc.alias, Path: validConjurPath}}

		// Run test case
		_, err := FileTemplateForFormat(fileFormat, secretSpecs)

		// Check result
		tc.assert(t, err, desc)
	}
}
