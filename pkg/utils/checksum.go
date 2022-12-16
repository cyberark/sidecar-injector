package utils

import (
	"bytes"
	"crypto/sha256"
	"io"
)

type Checksum []byte

func FileChecksum(buf *bytes.Buffer) (Checksum, error) {
	hash := sha256.New()
	bufCopy := bytes.NewBuffer(buf.Bytes())
	if _, err := io.Copy(hash, bufCopy); err != nil {
		return nil, err
	}
	checksum := hash.Sum(nil)
	return checksum, nil
}

func ContentHasChanged(groupName string, newChecksum Checksum, prevChecksums map[string]Checksum) bool {
	if prevChecksum, exists := prevChecksums[groupName]; exists {
		if bytes.Equal(newChecksum, prevChecksum) {
			return false
		}
	}
	return true
}
