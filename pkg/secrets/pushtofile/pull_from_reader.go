package pushtofile

import (
	"io"
	"io/ioutil"
	"os"
)

type pullFromReaderFunc func(
	reader io.Reader,
) (string, error)

type openReadCloserFunc func(
	path string,
) (io.ReadCloser, error)

func openFileAsReadCloser(path string) (io.ReadCloser, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(reader), nil
}

func pullFromReader(
	reader io.Reader,
) (string, error) {
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
