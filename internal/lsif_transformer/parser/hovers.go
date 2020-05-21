package parser

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const OffsetChunkSize = 8

type Offset struct {
	At  uint32
	Len uint32
}

type Hovers struct {
	File          *os.File
	IndexFile     *os.File
	CurrentOffset uint32
}

type RawResult struct {
	Contents []json.RawMessage `json:"contents"`
}

type RawData struct {
	Id     FlexInt   `json:"id"`
	Result RawResult `json:"result"`
}

type HoverRef struct {
	ResultSetId FlexInt `json:"outV"`
	HoverId     FlexInt `json:"inV"`
}

type ResultSetRef struct {
	ResultSetId FlexInt `json:"outV"`
	RefId       FlexInt `json:"inV"`
}

func NewHovers(tempDir string) (*Hovers, error) {
	file, err := ioutil.TempFile(tempDir, "hovers")
	if err != nil {
		return nil, err
	}

	indexFile, err := ioutil.TempFile(tempDir, "hovers-indexes")
	if err != nil {
		return nil, err
	}

	return &Hovers{
		File:          file,
		IndexFile:     indexFile,
		CurrentOffset: 0,
	}, nil
}

func (h *Hovers) Read(label string, line []byte) error {
	switch label {
	case "hoverResult":
		if err := h.addData(line); err != nil {
			return err
		}
	case "textDocument/hover":
		if err := h.addHoverRef(line); err != nil {
			return err
		}
	case "textDocument/references":
		if err := h.addResultSetRef(line); err != nil {
			return err
		}
	}

	return nil
}

func (h *Hovers) For(refId FlexInt) json.RawMessage {
	var offset Offset
	if err := ReadChunks(h.IndexFile, int64(refId*OffsetChunkSize), &offset); err != nil || offset.Len == 0 {
		return nil
	}

	hover := make([]byte, offset.Len)
	_, err := h.File.ReadAt(hover, int64(offset.At))
	if err != nil {
		return nil
	}

	return json.RawMessage(hover)
}

func (h *Hovers) Close() error {
	if err := h.File.Close(); err != nil {
		return err
	}

	if err := h.IndexFile.Close(); err != nil {
		return err
	}

	if err := os.Remove(h.IndexFile.Name()); err != nil {
		return err
	}

	return os.Remove(h.File.Name())
}

func (h *Hovers) addData(line []byte) error {
	var rawData RawData
	if err := json.Unmarshal(line, &rawData); err != nil {
		return err
	}

	codeHovers := []*CodeHover{}
	for _, rawContent := range rawData.Result.Contents {
		codeHover, err := NewCodeHover(rawContent)
		if err != nil {
			return err
		}

		codeHovers = append(codeHovers, codeHover)
	}

	codeHoversData, err := json.Marshal(codeHovers)
	if err != nil {
		return err
	}

	n, err := h.File.Write(codeHoversData)
	if err != nil {
		return err
	}

	l := uint32(n)
	offset := Offset{At: h.CurrentOffset, Len: l}
	h.CurrentOffset += l

	return WriteChunks(h.IndexFile, int64(rawData.Id*OffsetChunkSize), &offset)
}

func (h *Hovers) addHoverRef(line []byte) error {
	var hoverRef HoverRef
	if err := json.Unmarshal(line, &hoverRef); err != nil {
		return err
	}

	var offset Offset
	if err := ReadChunks(h.IndexFile, int64(hoverRef.HoverId*OffsetChunkSize), &offset); err != nil {
		return err
	}

	return WriteChunks(h.IndexFile, int64(hoverRef.ResultSetId*OffsetChunkSize), &offset)
}

func (h *Hovers) addResultSetRef(line []byte) error {
	var ref ResultSetRef
	if err := json.Unmarshal(line, &ref); err != nil {
		return err
	}

	var offset Offset
	if err := ReadChunks(h.IndexFile, int64(ref.ResultSetId*OffsetChunkSize), &offset); err != nil {
		return nil
	}

	return WriteChunks(h.IndexFile, int64(ref.RefId*OffsetChunkSize), &offset)
}
