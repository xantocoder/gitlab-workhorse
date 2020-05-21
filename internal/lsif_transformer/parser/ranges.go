package parser

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	Definitions    = "definitions"
	References     = "references"
	RangeChunkSize = 12
)

type Ranges struct {
	DefRefs map[FlexInt]*DefRef
	Hovers  *Hovers
	File    *os.File
}

type RawRange struct {
	Id   FlexInt `json:"id"`
	Data Range   `json:"start"`
}

type Range struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
	RefId     FlexInt
}

type RawDefRef struct {
	Property string    `json:"property"`
	RefId    FlexInt   `json:"outV"`
	RangeIds []FlexInt `json:"inVs"`
	DocId    FlexInt   `json:"document"`
}

type DefRef struct {
	Line  string
	DocId FlexInt
}

type SerializedRange struct {
	StartLine      uint32          `json:"start_line"`
	StartChar      uint32          `json:"start_char"`
	DefinitionPath string          `json:"definition_path,omitempty"`
	Hover          json.RawMessage `json:"hover"`
}

func NewRanges(tempDir string) (*Ranges, error) {
	hovers, err := NewHovers(tempDir)
	if err != nil {
		return nil, err
	}

	file, err := ioutil.TempFile(tempDir, "ranges")
	if err != nil {
		return nil, err
	}

	return &Ranges{
		DefRefs: make(map[FlexInt]*DefRef),
		Hovers:  hovers,
		File:    file,
	}, nil
}

func (r *Ranges) Read(label string, line []byte) error {
	switch label {
	case "range":
		if err := r.addRange(line); err != nil {
			return err
		}
	case "item":
		if err := r.addItem(line); err != nil {
			return err
		}
	default:
		return r.Hovers.Read(label, line)
	}

	return nil
}

func (r *Ranges) Serialize(f io.Writer, rangeIds []FlexInt, docs map[FlexInt]string) error {
	encoder := json.NewEncoder(f)
	n := len(rangeIds)

	if _, err := f.Write([]byte("[")); err != nil {
		return err
	}

	for i, rangeId := range rangeIds {
		entry, err := r.getRange(rangeId)
		if err != nil {
			continue
		}

		serializedRange := SerializedRange{
			StartLine:      entry.Line,
			StartChar:      entry.Character,
			DefinitionPath: r.definitionPathFor(docs, entry.RefId),
			Hover:          r.Hovers.For(entry.RefId),
		}
		if err := encoder.Encode(serializedRange); err != nil {
			return err
		}
		if i+1 < n {
			if _, err := f.Write([]byte(",")); err != nil {
				return err
			}
		}
	}

	if _, err := f.Write([]byte("]")); err != nil {
		return err
	}

	return nil
}

func (r *Ranges) Close() error {
	if err := r.File.Close(); err != nil {
		return err
	}

	if err := os.Remove(r.File.Name()); err != nil {
		return err
	}

	return r.Hovers.Close()
}

func (r *Ranges) definitionPathFor(docs map[FlexInt]string, refId FlexInt) string {
	defRef, ok := r.DefRefs[refId]
	if !ok {
		return ""
	}

	defPath := docs[defRef.DocId] + "#L" + defRef.Line

	return defPath
}

func (r *Ranges) addRange(line []byte) error {
	var rg RawRange
	if err := json.Unmarshal(line, &rg); err != nil {
		return err
	}

	offset := int64(rg.Id * RangeChunkSize)
	return WriteChunks(r.File, offset, &rg.Data)
}

func (r *Ranges) addItem(line []byte) error {
	var defRef RawDefRef
	if err := json.Unmarshal(line, &defRef); err != nil {
		return err
	}

	if defRef.Property != Definitions && defRef.Property != References {
		return nil
	}

	for _, rangeId := range defRef.RangeIds {
		offset := int64(rangeId*RangeChunkSize + 8)
		if err := WriteChunks(r.File, offset, &defRef.RefId); err != nil {
			return err
		}
	}

	if defRef.Property == Definitions {
		return r.addDefRef(&defRef)
	}

	return nil
}

func (r *Ranges) addDefRef(defRef *RawDefRef) error {
	offset := int64(defRef.RangeIds[0] * RangeChunkSize)
	var line uint32
	if err := ReadChunks(r.File, offset, &line); err != nil {
		return err
	}

	r.DefRefs[defRef.RefId] = &DefRef{
		Line:  strconv.Itoa(int(line + 1)),
		DocId: defRef.DocId,
	}

	return nil
}

func (r *Ranges) getRange(rangeId FlexInt) (*Range, error) {
	var rg Range
	if err := ReadChunks(r.File, int64(rangeId*RangeChunkSize), &rg); err != nil {
		return nil, err
	}

	return &rg, nil
}
