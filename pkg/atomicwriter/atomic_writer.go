package atomicwriter

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cyberark/conjur-authn-k8s-client/pkg/log"
	"github.com/cyberark/secrets-provider-for-k8s/pkg/log/messages"
)

// OS Function table
type osFuncs struct {
	chmod    func(string, os.FileMode) error
	rename   func(string, string) error
	remove   func(string) error
	sync     func(*os.File) error
	tempFile func(string, string) (*os.File, error)
	truncate func(string, int64) error
	write    func(*os.File, []byte) (int, error)
}

// Instantiation of OS Function table using std OS
var stdOSFuncs = osFuncs{
	chmod:  os.Chmod,
	rename: os.Rename,
	remove: os.Remove,
	sync: func(file *os.File) error {
		return file.Sync()
	},
	tempFile: ioutil.TempFile,
	truncate: os.Truncate,
	write: func(file *os.File, content []byte) (int, error) {
		return file.Write(content)
	},
}

type atomicWriter struct {
	path        string
	permissions os.FileMode
	tempFile    *os.File
	os          osFuncs
}

// NewAtomicWriter provides a simple atomic file writer which implements the
// io.WriteCloser interface. This allows us to use the atomic writer the way we
// would use any other Writer, such as a Buffer. Additonally, this struct
// takes the file path during construction, so the code which calls
// `Write()` doesn't need to be concerned with the destination, just like
// any other writer.
func NewAtomicWriter(path string, permissions os.FileMode) io.WriteCloser {
	return newAtomicWriter(path, permissions, stdOSFuncs)
}

func newAtomicWriter(path string, permissions os.FileMode, osFuncs osFuncs) io.WriteCloser {
	return &atomicWriter{
		path:        path,
		tempFile:    nil,
		permissions: permissions,
		os:          osFuncs,
	}
}

func (w *atomicWriter) Write(content []byte) (n int, err error) {
	// Create a temporary file if not created already
	if w.tempFile == nil {
		dir, file := filepath.Split(w.path)

		f, err := w.os.tempFile(dir, file)
		if err != nil {
			log.Error(messages.CSPFK055E, w.path)
			return 0, err
		}
		w.tempFile = f
	}

	// Write to the temporary file
	n, err = w.os.write(w.tempFile, content)
	if err != nil {
		log.Error(messages.CSPFK061E, w.path)
		w.Cleanup()
	}
	return n, err
}

func (w *atomicWriter) Close() error {
	if w.tempFile == nil {
		return nil
	}
	defer w.Cleanup()

	// Flush and close the temporary file
	err := w.os.sync(w.tempFile)
	if err != nil {
		log.Error(messages.CSPFK056E, w.tempFile.Name())
		return err
	}
	w.tempFile.Close()

	// Set the file permissions
	err = w.os.chmod(w.tempFile.Name(), w.permissions)
	if err != nil {
		log.Error(messages.CSPFK057E, w.tempFile.Name())
		// Try to rename the file anyway
	}

	// Rename the temporary file to the destination
	err = w.os.rename(w.tempFile.Name(), w.path)
	if err != nil {
		log.Error(messages.CSPFK058E, w.tempFile.Name(), w.path)
		return err
	}
	w.tempFile = nil

	return nil
}

// Cleanup attempts to remove the temporary file. This function is called by
// the `Close()` method, but can also be called manually in cases where `Close()`
// is not called.
func (w *atomicWriter) Cleanup() {
	if w.tempFile == nil {
		return
	}

	err := w.os.remove(w.tempFile.Name())
	if err == nil || os.IsNotExist(err) {
		w.tempFile = nil
		return
	}

	// If we can't remove the temporary directory, truncate the file to remove all secret content
	err = w.os.truncate(w.tempFile.Name(), 0)
	switch {
	case os.IsNotExist(err):
		// This shouldn't happen, but just to be safe.
		w.tempFile = nil
	case err != nil:
		// Truncate failed as well
		log.Error(messages.CSPFK060E, w.tempFile.Name(), w.path)
	default:
		log.Error(messages.CSPFK059E, w.tempFile.Name(), w.path)
	}
}
