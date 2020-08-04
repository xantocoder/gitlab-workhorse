package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferencesStore(t *testing.T) {
	r, err := NewReferences(Config{ProcessReferences: true})
	require.NoError(t, err)

	err = r.Store(3, []Item{{Line: 2, DocId: 1}, {Line: 3, DocId: 1}})
	require.NoError(t, err)

	docs := map[Id]string{1: "doc.go"}
	serializedReferences := r.For(docs, 3)

	require.Contains(t, serializedReferences, SerializedReference{Path: "doc.go#L2"})
	require.Contains(t, serializedReferences, SerializedReference{Path: "doc.go#L3"})

	require.NoError(t, r.Close())
}

func TestReferencesStoreEmpty(t *testing.T) {
	r, err := NewReferences(Config{ProcessReferences: true})
	require.NoError(t, err)

	err = r.Store(3, []Item{})
	require.NoError(t, err)

	docs := map[Id]string{1: "doc.go"}
	serializedReferences := r.For(docs, 3)

	require.Nil(t, serializedReferences)
	require.NoError(t, r.Close())
}
