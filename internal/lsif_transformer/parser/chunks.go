package parser

import (
	"bytes"
	"encoding/binary"
	"os"
)

func WriteChunks(f *os.File, offset int64, data interface{}) error {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)
	if err != nil {
		return err
	}

	_, err = f.WriteAt(buf.Bytes(), offset)
	return err
}

func ReadChunks(f *os.File, offset int64, data interface{}) error {
	b := make([]byte, binary.Size(data))
	if n, err := f.ReadAt(b, offset); err != nil {
		if n == 0 {
			return err
		}
	}

	return binary.Read(bytes.NewReader(b), binary.LittleEndian, data)
}
