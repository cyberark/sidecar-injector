package secrets

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testChmod returns a function that wraps the standard os.Chmod() function,
// with an option to return a configurable "injected" error for testing.
func testChmod(injectErr error) chmodFunc {
	return func(path string, mode os.FileMode) error {
		if injectErr != nil {
			return injectErr
		}
		return stdOSFuncs.chmod(path, mode)
	}
}

// testCreate returns a function that wraps the standard os.Create() function,
// with an option to return a configurable "injected" error for testing.
func testCreate(injectErr error) createFunc {
	return func(path string) (*os.File, error) {
		if injectErr != nil {
			return nil, injectErr
		}
		return stdOSFuncs.create(path)
	}
}

// testOpen returns a function that wraps the standard os.Open() function,
// with an option to return a configurable "injected" error for testing.
func testOpen(injectErr error) openFunc {
	return func(path string) (*os.File, error) {
		if injectErr != nil {
			return nil, injectErr
		}
		return stdOSFuncs.open(path)
	}
}

func testMkdirAll(injectErr error) mkdirAllFunc {
	return func(path string, mode os.FileMode) error {
		if injectErr != nil {
			return injectErr
		}
		return stdOSFuncs.mkdirAll(path, mode)
	}
}

// injectErrs defines a set of errors that should be injected for a given
// test case.
type injectErrs struct {
	chmodErr  error
	createErr error
	openErr   error
	mkDirErr  error
}

// testOSFuncs generates a set of OS functions for testing, each of which
// support an option to return an "injected" error.
func testOSFuncs(inject injectErrs) osFuncs {
	return osFuncs{
		chmod:    testChmod(inject.chmodErr),
		create:   testCreate(inject.createErr),
		open:     testOpen(inject.openErr),
		mkdirAll: testMkdirAll(inject.mkDirErr),
	}
}

// testStatusUpdater wraps a fileUpdater that uses temp files for
// recording provider status.
type testStatusUpdater struct {
	tempDir          string
	fileUpdater      fileUpdater
	targetScriptFile string
}

func newTestStatusUpdater(injectErrs injectErrs) (*testStatusUpdater, error) {
	tempDir, err := ioutil.TempDir("", "secrets-testing")
	if err != nil {
		return nil, err
	}

	var tempFile = func(filename string) string {
		return filepath.Join(tempDir, filename)
	}

	updater := &testStatusUpdater{}
	updater.tempDir = tempDir
	updater.fileUpdater = fileUpdater{
		providedFile:  tempFile("CONJUR_SECRETS_PROVIDED"),
		updatedFile:   tempFile("CONJUR_SECRETS_UPDATED"),
		scripts:       []string{"conjur-secrets-unchanged.sh"},
		scriptSrcDir:  "../../bin/run-time-scripts",
		scriptDestDir: tempDir,
		os:            testOSFuncs(injectErrs),
	}
	updater.targetScriptFile = tempFile("conjur-secrets-unchanged.sh")
	return updater, nil
}

func (updater testStatusUpdater) cleanup() {
	os.RemoveAll(updater.tempDir)
}

func TestSetStatus(t *testing.T) {
	TestCases := []struct {
		description string
		injectErrs  injectErrs
		runTwice    bool
		assert      func(*testing.T, string, error)
	}{
		{
			description: "Happy path",
			assert: func(t *testing.T, statusFile string, err error) {
				assert.NoError(t, err)
				info, err := os.Stat(statusFile)
				assert.NoError(t, err)
				assert.Equal(t, info.Mode(), os.FileMode(statusFileMode))
			},
		},
		{
			description: "Happy path, set status twice",
			runTwice:    true,
			assert: func(t *testing.T, statusFile string, err error) {
				assert.NoError(t, err)
				info, err := os.Stat(statusFile)
				assert.NoError(t, err)
				assert.Equal(t, info.Mode(), os.FileMode(statusFileMode))
			},
		},
		{
			description: "Error on status file create",
			injectErrs:  injectErrs{createErr: os.ErrPermission},
			assert: func(t *testing.T, statusFile string, err error) {
				assert.Error(t, err)
				assert.True(t, os.IsPermission(err))
				assert.NoFileExists(t, statusFile)
			},
		},
		{
			description: "Error on status file chmod",
			injectErrs:  injectErrs{chmodErr: os.ErrPermission},
			assert: func(t *testing.T, statusFile string, err error) {
				assert.Error(t, err)
				assert.True(t, os.IsPermission(err))
				assert.FileExists(t, statusFile)
			},
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.description, func(t *testing.T) {
			// Set up test
			updater, err := newTestStatusUpdater(tc.injectErrs)
			assert.NoError(t, err)
			defer updater.cleanup()

			// Run test
			fileUpdater := updater.fileUpdater
			statusFile := fileUpdater.providedFile
			err = fileUpdater.setStatus(statusFile)
			if tc.runTwice {
				assert.NoError(t, err)
				err = fileUpdater.setStatus(statusFile)
			}

			// Check results
			tc.assert(t, statusFile, err)
		})
	}
}

func TestCopyScripts(t *testing.T) {
	TestCases := []struct {
		description string
		injectErrs  injectErrs
		runTwice    bool
		assert      func(*testing.T, string, error)
	}{
		{
			description: "Happy path",
			assert: func(t *testing.T, scriptFile string, err error) {
				assert.NoError(t, err)
				info, err := os.Stat(scriptFile)
				assert.NoError(t, err)
				assert.Equal(t, info.Mode(), os.FileMode(scriptFileMode))
			},
		},
		{
			description: "Happy path, copy scripts twice",
			runTwice:    true,
			assert: func(t *testing.T, scriptFile string, err error) {
				assert.NoError(t, err)
				info, err := os.Stat(scriptFile)
				assert.NoError(t, err)
				assert.Equal(t, info.Mode(), os.FileMode(scriptFileMode))
			},
		},
		{
			description: "Error on source file open",
			injectErrs:  injectErrs{openErr: os.ErrPermission},
			assert: func(t *testing.T, scriptFile string, err error) {
				assert.Error(t, err)
				assert.True(t, os.IsPermission(err))
				assert.NoFileExists(t, scriptFile)
			},
		},
		{
			description: "Error on destination file create",
			injectErrs:  injectErrs{createErr: os.ErrPermission},
			assert: func(t *testing.T, scriptFile string, err error) {
				assert.Error(t, err)
				assert.True(t, os.IsPermission(err))
				assert.NoFileExists(t, scriptFile)
			},
		},
		{
			description: "Error on script file chmod",
			injectErrs:  injectErrs{chmodErr: os.ErrPermission},
			assert: func(t *testing.T, scriptFile string, err error) {
				assert.Error(t, err)
				assert.True(t, os.IsPermission(err))
				assert.FileExists(t, scriptFile)
			},
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.description, func(t *testing.T) {
			// Set up test
			updater, err := newTestStatusUpdater(tc.injectErrs)
			assert.NoError(t, err)
			defer updater.cleanup()

			// Run test
			fileUpdater := updater.fileUpdater
			err = fileUpdater.copyScripts()
			if tc.runTwice {
				assert.NoError(t, err)
				err = fileUpdater.copyScripts()
			}

			// Check results
			tc.assert(t, updater.targetScriptFile, err)
		})
	}
}
