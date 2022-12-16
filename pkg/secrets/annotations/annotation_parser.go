package annotations

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

// fileOpener is a function type that captures dependency injection for
// filesystem operations. It returns an instantiation of the 'io.ReadCloser'
// interface, which incorporates the only two filesystem operations that we
// need for parsing an annotations file:
//   - File closer
//   - IO reader
type fileOpener func(name string, flag int, perm os.FileMode) (io.ReadCloser, error)

// osFileOpener is a 'fileOpener' that uses standard OS.
func osFileOpener(name string, flag int, perm os.FileMode) (io.ReadCloser, error) {
	return os.OpenFile(name, flag, perm)
}

// NewAnnotationsFromFile reads and parses an annotations file that has been
// created by Kubernetes via the Downward API, based on Pod annotations that
// are defined in a deployment manifest.
func NewAnnotationsFromFile(path string) (map[string]string, error) {
	// Use standard OS
	res, err := newAnnotationsFromFile(osFileOpener, path)
	if err != nil {
		return nil, fmt.Errorf(messages.CSPFK041E, path, err)
	}

	return res, nil
}

// newAnnotationsFromFile performs the work of NewAnnotationsFromFile(), and
// provides a function entrypoint that allows filesystem mocking for test
// purposes.
func newAnnotationsFromFile(fo fileOpener, path string) (map[string]string, error) {
	annotationsFile, err := fo(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer annotationsFile.Close()
	return newAnnotationsFromReader(annotationsFile)
}

// newAnnotationsFromReader parses an input stream representing an annotations file that
// had been created by Kubernetes via the Downward API, returning a
// string-to-string map of annotations key-value pairs.
//
// List and multi-line annotations are formatted as a single string in the
// annotations file, and this format persists into the map returned by this
// function. For example, the following annotation:
//   conjur.org/conjur-secrets.cache: |
//     - url
//     - admin-password: password
//     - admin-username: username
// Is stored in the annotations file as:
//   conjur.org/conjur-secrets.cache="- url\n- admin-password: password\n- admin-username: username\n"
func newAnnotationsFromReader(annotationsFile io.Reader) (map[string]string, error) {
	var lines []string
	scanner := bufio.NewScanner(annotationsFile)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	annotationsMap := make(map[string]string)
	for lineNumber, line := range lines {
		keyValuePair := strings.SplitN(line, "=", 2)
		if len(keyValuePair) == 1 {
			return nil, log.RecordedError(messages.CSPFK045E, lineNumber+1)
		}

		key := keyValuePair[0]
		value, err := strconv.Unquote(keyValuePair[1])
		if err != nil {
			return nil, log.RecordedError(messages.CSPFK045E, lineNumber+1)
		}

		annotationsMap[key] = value
	}

	return annotationsMap, nil
}
