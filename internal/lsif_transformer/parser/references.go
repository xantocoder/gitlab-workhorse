package parser

import (
	"strconv"
)

type References struct {
	File              *dynamicCache
	Offsets           *cache
	ProcessReferences bool
}

type SerializedReference struct {
	Path string `json:"path"`
}

func NewReferences(config Config) (*References, error) {
	tempPath := config.TempPath

	file, err := newDynamicCache(tempPath, "references")
	if err != nil {
		return nil, err
	}

	offsets, err := newCache(tempPath, "references-offsets", Offset{})
	if err != nil {
		return nil, err
	}

	return &References{
		File:              file,
		Offsets:           offsets,
		ProcessReferences: config.ProcessReferences,
	}, nil
}

func (r *References) Store(refId Id, references []Item) error {
	size := len(references)

	if !r.ProcessReferences || size == 0 {
		return nil
	}

	err := r.File.SetEntry(refId, references)
	if err != nil {
		return err
	}

	r.Offsets.SetEntry(refId, Offset{Len: int32(size)})

	return nil
}

func (r *References) For(docs map[Id]string, refId Id) []SerializedReference {
	if !r.ProcessReferences {
		return nil
	}

	var offset Offset
	if err := r.Offsets.Entry(refId, &offset); err != nil || offset.Len == 0 {
		return nil
	}

	references := make([]Item, offset.Len)
	if err := r.File.Entry(refId, &references); err != nil {
		return nil
	}

	var serializedReferences []SerializedReference

	for _, reference := range references {
		serializedReference := SerializedReference{
			Path: docs[reference.DocId] + "#L" + strconv.Itoa(int(reference.Line)),
		}

		serializedReferences = append(serializedReferences, serializedReference)
	}

	return serializedReferences
}

func (r *References) Close() error {
	return combineErrors(
		r.File.Close(),
		r.Offsets.Close(),
	)
}
