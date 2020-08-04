package parser

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

type chunk struct {
	A int16
	B int16
}

func TestCache(t *testing.T) {
	cache, err := newCache("", "test-chunks", chunk{})
	require.NoError(t, err)
	defer cache.Close()

	c := chunk{A: 1, B: 2}
	require.NoError(t, cache.SetEntry(1, &c))
	require.NoError(t, cache.setOffset(0))

	content, err := ioutil.ReadAll(cache.file)
	require.NoError(t, err)

	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x2, 0x0}
	require.Equal(t, expected, content)

	var nc chunk
	require.NoError(t, cache.Entry(1, &nc))
	require.Equal(t, c, nc)
}

func TestDynamicCache(t *testing.T) {
	cache, err := newDynamicCache("", "test-chunks")
	require.NoError(t, err)
	defer cache.Close()

	c := []chunk{{A: 1, B: 2}, {A: 1, B: 2}}
	require.NoError(t, cache.SetEntry(1, c))
	require.NoError(t, cache.SetEntry(2, c))

	nc := make([]chunk, len(c))
	require.NoError(t, cache.Entry(2, &nc))
	require.Equal(t, c, nc)
}
