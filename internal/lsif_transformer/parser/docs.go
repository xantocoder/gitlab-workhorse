package parser

import (
	"archive/zip"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

type Line struct {
	Type string `json:"label"`
}

type Docs struct {
	Root          string
	Entries       map[FlexInt]string
	DocRanges     map[FlexInt]*Offset
	Ranges        *Ranges
	File          *os.File
	CurrentOffset uint32
}

type Document struct {
	Id  FlexInt `json:"id"`
	Uri string  `json:"uri"`
}

type DocumentRange struct {
	OutV     FlexInt   `json:"outV"`
	RangeIds []FlexInt `json:"inVs"`
}

type Metadata struct {
	Root string `json:"projectRoot"`
}

func NewDocs(tempDir string) (*Docs, error) {
	file, err := ioutil.TempFile(tempDir, "docs")
	if err != nil {
		return nil, err
	}

	ranges, err := NewRanges(tempDir)
	if err != nil {
		return nil, err
	}

	return &Docs{
		Root:          "file:///",
		Entries:       make(map[FlexInt]string),
		DocRanges:     make(map[FlexInt]*Offset),
		Ranges:        ranges,
		File:          file,
		CurrentOffset: 0,
	}, nil
}

func (d *Docs) Read(line []byte) error {
	l := Line{}
	if err := json.Unmarshal(line, &l); err != nil {
		return err
	}

	switch l.Type {
	case "metaData":
		if err := d.addMetadata(line); err != nil {
			return err
		}
	case "document":
		if err := d.addDocument(line); err != nil {
			return err
		}
	case "contains":
		if err := d.addDocRanges(line); err != nil {
			return err
		}
	default:
		return d.Ranges.Read(l.Type, line)
	}

	return nil
}

func (d *Docs) Close() error {
	if err := d.File.Close(); err != nil {
		return err
	}

	if err := os.Remove(d.File.Name()); err != nil {
		return err
	}

	return d.Ranges.Close()
}

func (d *Docs) SerializeEntries(w *zip.Writer) error {
	for id, path := range d.Entries {
		filePath := Lsif + "/" + path + ".json"

		f, err := w.Create(filePath)
		if err != nil {
			return err
		}

		ranges, err := d.getRangeIds(id)
		if err != nil {
			return err
		}

		if err := d.Ranges.Serialize(f, ranges, d.Entries); err != nil {
			return err
		}
	}

	return nil
}

func (d *Docs) addMetadata(line []byte) error {
	var metadata Metadata
	if err := json.Unmarshal(line, &metadata); err != nil {
		return err
	}

	d.Root = strings.TrimSpace(metadata.Root) + "/"

	return nil
}

func (d *Docs) addDocument(line []byte) error {
	var doc Document
	if err := json.Unmarshal(line, &doc); err != nil {
		return err
	}

	d.Entries[doc.Id] = strings.TrimPrefix(doc.Uri, d.Root)

	return nil
}

func (d *Docs) addDocRanges(line []byte) error {
	var docRange DocumentRange
	if err := json.Unmarshal(line, &docRange); err != nil {
		return err
	}

	ranges := docRange.RangeIds
	n, err := WriteChunks(d.File, int64(d.CurrentOffset), &ranges)
	if err != nil {
		return err
	}

	d.DocRanges[docRange.OutV] = &Offset{At: d.CurrentOffset, Len: uint32(len(ranges))}
	d.CurrentOffset += uint32(n)

	return nil
}

func (d *Docs) getRangeIds(id FlexInt) ([]FlexInt, error) {
	offset, ok := d.DocRanges[id]
	if !ok || offset == nil {
		return []FlexInt{}, nil
	}

	ranges := make([]FlexInt, offset.Len)
	if err := ReadChunks(d.File, int64(offset.At), &ranges); err != nil {
		return []FlexInt{}, err
	}

	return ranges, nil
}
