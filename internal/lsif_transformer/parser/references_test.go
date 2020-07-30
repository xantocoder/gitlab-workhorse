package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferencesStore(t *testing.T) {
	r := NewReferences(Config{ProcessReferences: true})

	r.Store(3, []Item{{Line: 2, DocId: 1}, {Line: 3, DocId: 1}})

	docs := map[Id]string{1: "doc.go"}
	serializedReferences := r.For(docs, 3)

	require.Contains(t, serializedReferences, SerializedReference{Path: "doc.go#L2"})
	require.Contains(t, serializedReferences, SerializedReference{Path: "doc.go#L3"})
}
