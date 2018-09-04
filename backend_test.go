package gitobj

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryBackend(t *testing.T) {
	backend, err := NewMemoryBackend(nil)
	assert.NoError(t, err)

	ro, rw := backend.Storage()
	assert.Equal(t, ro, rw)
	assert.NotNil(t, ro.(*memoryStorer))
}

func TestNewMemoryBackendWithReadOnlyData(t *testing.T) {
	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	oid, err := hex.DecodeString(sha)

	assert.Nil(t, err)

	m := map[string]io.ReadWriter{
		sha: bytes.NewBuffer([]byte{0x1}),
	}

	backend, err := NewMemoryBackend(m)
	assert.NoError(t, err)

	ro, _ := backend.Storage()
	reader, err := ro.Open(oid)
	assert.NoError(t, err)

	contents, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1}, contents)
}

func TestNewMemoryBackendWithWritableData(t *testing.T) {
	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	oid, err := hex.DecodeString(sha)

	assert.Nil(t, err)

	backend, err := NewMemoryBackend(make(map[string]io.ReadWriter))
	assert.NoError(t, err)

	buf := bytes.NewBuffer([]byte{0x1})

	ro, rw := backend.Storage()
	rw.Store(oid, buf)

	reader, err := ro.Open(oid)
	assert.NoError(t, err)

	contents, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1}, contents)
}
