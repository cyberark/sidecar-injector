package utils

import (
	"bytes"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChecksum(t *testing.T) {
	t.Run("", func(t *testing.T) {
		file := []byte("environment: prod\n" +
			"url: https://example.com\n" +
			"username: example-user\n" +
			"password: example-pass\n")
		expected := "e218dd95515fe368b6befe4b39f381fb82391d4f11b690225f27c9fcfe60d78b"
		t.Run("Happy Case for FileChecksum", func(t *testing.T) {
			b:=bytes.NewBuffer(file)
			checksum, err := FileChecksum(b)
			encodedChecksum := hex.EncodeToString(checksum)
			assert.Equal(t, expected, encodedChecksum)
			assert.NoError(t, err)
		})
		t.Run("No change case for ContentHasChanged", func(t *testing.T) {
			prevChecksums := map[string]Checksum{}
			group := "testGroup"
			checksum1 := []byte("e218dd95515fe368b6befe4b39f381fb")
			checksum2 := []byte("82391d4f11b690225f27c9fcfe60d78b")
			prevChecksums[group] = checksum1
			prevChecksums[group + "2"] = checksum2
			assert.False(t, ContentHasChanged(group, checksum1, prevChecksums))
		})
		t.Run("Change case for ContentHasChanged", func(t *testing.T) {
			prevChecksums := map[string]Checksum{}
			group := "testGroup"
			checksum1 := []byte("e218dd95515fe368b6befe4b39f381fb")
			checksum2 := []byte("82391d4f11b690225f27c9fcfe60d78b")
			prevChecksums[group] = checksum1
			prevChecksums[group + "2"] = checksum2
			assert.True(t, ContentHasChanged(group, []byte("abcddd95515fe368b6befe4b39f381fb"), prevChecksums))
		})
	})
}
