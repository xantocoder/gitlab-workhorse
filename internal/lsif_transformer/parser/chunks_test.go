package parser

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type Chunk struct {
	A uint16
	B uint16
}

func TestWriteChunks(t *testing.T) {
	f, err := ioutil.TempFile("", "test-chunks")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	c := Chunk{A: 1, B: 2}
	n, err := WriteChunks(f, 2, &c)
	require.Equal(t, 4, n)
	require.NoError(t, err)

	content, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, []byte{0x0, 0x0, 0x1, 0x0, 0x2, 0x0}, content)
}

func TestReadChunks(t *testing.T) {
	f, err := ioutil.TempFile("", "test-chunks")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	r := Range{Line: 100, Character: 123}

	n, err := WriteChunks(f, 12, &r)
	require.NoError(t, err)
	require.Equal(t, 12, n)

	n, err = WriteChunks(f, 20, FlexInt(234))
	require.NoError(t, err)
	require.Equal(t, 4, n)

	var rg Range
	require.NoError(t, ReadChunks(f, 12, &rg))

	expected := Range{Line: 100, Character: 123, RefId: 234}
	require.Equal(t, expected, rg)
}
