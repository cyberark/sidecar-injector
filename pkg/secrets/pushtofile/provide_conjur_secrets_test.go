package pushtofile

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cyberark/secrets-provider-for-k8s/pkg/secrets/clients/conjur"
	"github.com/stretchr/testify/assert"
)

func retrieve(variableIDs []string, ctx context.Context) (map[string][]byte, error) {
	masterMap := make(map[string][]byte)
	for _, id := range variableIDs {
		masterMap[id] = []byte(fmt.Sprintf("value-%s", id))
	}
	return masterMap, nil
}

func retrieveWith403(variableIDs []string, ctx context.Context) (map[string][]byte, error) {
	return nil, fmt.Errorf("403")
}

func retrieveWithGenericError(variableIDs []string, ctx context.Context) (map[string][]byte, error) {
	return nil, fmt.Errorf("generic error")
}

func secretGroups(filePath string) []*SecretGroup {
	return []*SecretGroup{
		{
			Name:            "groupname",
			FilePath:        filePath,
			FileFormat:      "yaml",
			FilePermissions: 123,
			SecretSpecs: []SecretSpec{
				{
					Alias: "password",
					Path:  "path1",
				},
			},
		},
	}
}

func TestNewProvider(t *testing.T) {
	TestCases := []struct {
		description         string
		retrieveFunc        conjur.RetrieveSecretsFunc
		basePath            string
		annotations         map[string]string
		expectedSecretGroup []*SecretGroup
	}{
		{
			description:  "happy case",
			retrieveFunc: retrieve,
			basePath:     "/basepath",
			annotations: map[string]string{
				"conjur.org/conjur-secrets.groupname":     "- password: path1\n",
				"conjur.org/secret-file-path.groupname":   "path/to/file",
				"conjur.org/secret-file-format.groupname": "yaml",
			},
			expectedSecretGroup: []*SecretGroup{
				{
					Name:            "groupname",
					FilePath:        "/basepath/path/to/file",
					FileTemplate:    "",
					FileFormat:      "yaml",
					FilePermissions: defaultFilePermissions,
					SecretSpecs: []SecretSpec{
						{
							Alias: "password",
							Path:  "path1",
						},
					},
				},
			},
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.description, func(t *testing.T) {
			config := P2FProviderConfig{
				SecretFileBasePath:   tc.basePath,
				TemplateFileBasePath: "",
				AnnotationsMap:       tc.annotations,
			}
			p, err := NewProvider(tc.retrieveFunc, false, config)
			assert.Empty(t, err)
			assert.Equal(t, tc.expectedSecretGroup, p.secretGroups)
		})
	}
}

func TestProvideWithDeps(t *testing.T) {
	TestCases := []struct {
		description     string
		provider        fileProvider
		createFileName  string
		sanitizeEnabled bool
		assert          func(*testing.T, fileProvider, bool, error, *ClosableBuffer, pushToWriterSpy, openWriteCloserSpy)
	}{
		{
			description: "happy path",
			provider: fileProvider{
				retrieveSecretsFunc: retrieve,
				secretGroups:        secretGroups("/path/to/file"),
			},
			sanitizeEnabled: true,
			assert: func(
				t *testing.T,
				p fileProvider,
				updated bool,
				err error,
				closableBuf *ClosableBuffer,
				spyPushToWriter pushToWriterSpy,
				spyOpenWriteCloser openWriteCloserSpy,
			) {
				assert.True(t, updated)
				assert.Equal(t, closableBuf, spyPushToWriter.args.writer)
				assert.Equal(t, spyOpenWriteCloser.args.path, p.secretGroups[0].FilePath)
				assert.Nil(t, err)
			},
		},
		{
			description:    "403 error",
			createFileName: "path_to_file.yaml",
			provider: fileProvider{
				retrieveSecretsFunc: retrieveWith403,
				secretGroups:        secretGroups("path_to_file.yaml"),
			},
			sanitizeEnabled: true,
			assert: func(
				t *testing.T,
				p fileProvider,
				updated bool,
				err error,
				closableBuf *ClosableBuffer,
				spyPushToWriter pushToWriterSpy,
				spyOpenWriteCloser openWriteCloserSpy,
			) {
				assert.True(t, updated)
				assert.Error(t, err)
				// File should be deleted because of 403 error
				assert.NoFileExists(t, "path_to_file.yaml")
			},
		},
		{
			description:    "generic error",
			createFileName: "path_to_file.yaml",
			provider: fileProvider{
				retrieveSecretsFunc: retrieveWithGenericError,
				secretGroups:        secretGroups("path_to_file.yaml"),
			},
			sanitizeEnabled: true,
			assert: func(
				t *testing.T,
				p fileProvider,
				updated bool,
				err error,
				closableBuf *ClosableBuffer,
				spyPushToWriter pushToWriterSpy,
				spyOpenWriteCloser openWriteCloserSpy,
			) {
				assert.False(t, updated)
				assert.Error(t, err)
				// File should not be deleted because of generic error
				assert.FileExists(t, "path_to_file.yaml")
			},
		},
		{
			description:    "403 error with sanitize disabled",
			createFileName: "path_to_file.yaml",
			provider: fileProvider{
				retrieveSecretsFunc: retrieveWith403,
				secretGroups:        secretGroups("path_to_file.yaml"),
			},
			sanitizeEnabled: false,
			assert: func(
				t *testing.T,
				p fileProvider,
				updated bool,
				err error,
				closableBuf *ClosableBuffer,
				spyPushToWriter pushToWriterSpy,
				spyOpenWriteCloser openWriteCloserSpy,
			) {
				assert.False(t, updated)
				assert.Error(t, err)
				// File shouldn't be deleted because sanitize is disabled
				assert.FileExists(t, "path_to_file.yaml")
			},
		},
	}

	for _, tc := range TestCases {
		t.Run(tc.description, func(t *testing.T) {
			// Setup mocks
			closableBuf := new(ClosableBuffer)
			spyPushToWriter := pushToWriterSpy{
				targetsUpdated: true,
			}
			spyOpenWriteCloser := openWriteCloserSpy{
				writeCloser: closableBuf,
			}

			if tc.createFileName != "" {
				_, err := os.Create(tc.createFileName)
				assert.NoError(t, err)
				defer os.Remove(tc.createFileName)
			}

			updated, err := provideWithDeps(
				context.Background(),
				tc.provider.secretGroups,
				tc.sanitizeEnabled,
				fileProviderDepFuncs{
					retrieveSecretsFunc: tc.provider.retrieveSecretsFunc,
					depOpenWriteCloser:  spyOpenWriteCloser.Call,
					depPushToWriter:     spyPushToWriter.Call,
				},
			)

			tc.assert(t, tc.provider, updated, err, closableBuf, spyPushToWriter, spyOpenWriteCloser)
		})
	}
}
