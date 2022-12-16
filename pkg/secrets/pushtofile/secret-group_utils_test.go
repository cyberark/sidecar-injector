package pushtofile

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
)

type ClosableBuffer struct {
	bytes.Buffer
	CloseErr error
}

func (c ClosableBuffer) Close() error { return c.CloseErr }

//// pushToWriterFunc
type pushToWriterArgs struct {
	writer        io.Writer
	groupName     string
	groupTemplate string
	groupSecrets  []*Secret
}

type pushToWriterSpy struct {
	args           pushToWriterArgs
	targetsUpdated bool
	err            error
	_calls         int
}

func (spy *pushToWriterSpy) Call(
	writer io.Writer,
	groupName string,
	groupTemplate string,
	groupSecrets []*Secret,
) (bool, error) {
	spy._calls++
	// This is to ensure the spy is only ever used once!
	if spy._calls > 1 {
		panic("spy called more than once")
	}

	spy.args = pushToWriterArgs{
		writer:        writer,
		groupName:     groupName,
		groupTemplate: groupTemplate,
		groupSecrets:  groupSecrets,
	}

	return spy.targetsUpdated, spy.err
}

//// openWriteCloserFunc
type openWriteCloserArgs struct {
	path        string
	permissions os.FileMode
}

type openWriteCloserSpy struct {
	args        openWriteCloserArgs
	writeCloser io.WriteCloser
	err         error
	_calls      int
}

func (spy *openWriteCloserSpy) Call(path string, permissions os.FileMode) (io.WriteCloser, error) {
	spy._calls++
	// This is to ensure the spy is only ever used once!
	if spy._calls > 1 {
		panic("spy called more than once")
	}

	spy.args = openWriteCloserArgs{
		path:        path,
		permissions: permissions,
	}

	return spy.writeCloser, spy.err
}

//// pullFromReaderFunc
type pullFromReaderArgs struct {
	reader io.Reader
}

type pullFromReaderSpy struct {
	args   pullFromReaderArgs
	err    error
	_calls int
}

func (spy *pullFromReaderSpy) Call(
	reader io.Reader,
) (string, error) {
	spy._calls++
	// This is to ensure the spy is only ever used once!
	if spy._calls > 1 {
		panic("spy called more than once")
	}

	spy.args = pullFromReaderArgs{
		reader: reader,
	}

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return string(content), err
	}

	return string(content), spy.err
}

//// openReadCloserFunc
type openReadCloserArgs struct {
	path string
}

type openReadCloserSpy struct {
	args       openReadCloserArgs
	readCloser io.ReadCloser
	err        error
	_calls     int
}

func (spy *openReadCloserSpy) Call(path string) (io.ReadCloser, error) {
	spy._calls++
	// This is to ensure the spy is only ever used once!
	if spy._calls > 1 {
		panic("spy called more than once")
	}

	spy.args = openReadCloserArgs{
		path: path,
	}

	return spy.readCloser, spy.err
}
