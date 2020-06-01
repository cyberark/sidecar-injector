package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_gitVersionIsPresent(t *testing.T) {
	assert.NotEmpty(t, gitVersion, "Expected gitVersion to be non-empty but got an empty value")
}

func Test_gitCommitShortIsPresent(t *testing.T) {
	assert.NotEmpty(t, gitCommitShort, "Expected gitCommitShort to be non-empty but got an empty value")
}

func Test_gitVersionIsCorrectFormat(t *testing.T) {
	assert.Regexp(t, `^[0-9]+\.[0-9]+\.[0-9]+$`, gitVersion,
		"Expected gitVersion to be a SemVer string")
}

func TestGetReturnsCorrectFormat(t *testing.T) {
	assert.Regexp(t, `^[0-9]+\.[0-9]+\.[0-9]+-[a-z0-9]+$`, Get(),
		"Get should return a '<SemVer>-<alphanumeric>' string")
}

// For now we enforce just plain lowercase alphanumerics, which matches 'dev' and sha1
// hashes.
func Test_gitCommitShortIsCorrectFormat(t *testing.T) {
	assert.Regexp(t, `^[a-z0-9]+$`, gitCommitShort,
		"gitCommitShort should be a strict lowercase alphanumeric string")
}
