package pushtofile

import (
	"bytes"
	"testing"
)

type pullFromReaderTestCase struct {
	description string
	content     string
	assert      func(*testing.T, string, error)
}

func (tc pullFromReaderTestCase) Run(t *testing.T) {
	t.Run(tc.description, func(t *testing.T) {
		buf := bytes.NewBufferString(tc.content)
		readContent, err := pullFromReader(buf)
		tc.assert(t, readContent, err)
	})
}

var pullFromReaderTestCases = []pullFromReaderTestCase{
	{
		description: "happy case",
		content:     "template file content",
		assert:      assertGoodOutput("template file content"),
	},
}

func TestPullFromReader(t *testing.T) {
	for _, tc := range pullFromReaderTestCases {
		tc.Run(t)
	}
}
