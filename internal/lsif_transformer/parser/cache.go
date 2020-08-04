package parser

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
)

// This cache implementation is using a temp file to provide key-value data storage
// It allows to avoid storing intermediate calculations in RAM
// The stored data must be a fixed-size value or a slice of fixed-size values, or a pointer to such data
type cache struct {
	file      *os.File
	chunkSize int64
}

type dynamicCache struct {
	file *os.File
}

type Offset struct {
	At  int32
	Len int32
}

func newCache(tempDir, filename string, data interface{}) (*cache, error) {
	cacheFile, err := createCacheFile(tempDir, filename)
	if err != nil {
		return nil, err
	}

	return &cache{file: cacheFile, chunkSize: int64(binary.Size(data))}, nil
}

func newDynamicCache(tempDir, filename string) (*dynamicCache, error) {
	cacheFile, err := createCacheFile(tempDir, filename)
	if err != nil {
		return nil, err
	}

	return &dynamicCache{file: cacheFile}, nil
}

func createCacheFile(tempDir, filename string) (*os.File, error) {
	f, err := ioutil.TempFile(tempDir, filename)
	if err != nil {
		return nil, err
	}

	if err := os.Remove(f.Name()); err != nil {
		return nil, err
	}

	return f, nil
}

func (c *cache) SetEntry(id Id, data interface{}) error {
	if err := c.setOffset(id); err != nil {
		return err
	}

	return binary.Write(c.file, binary.LittleEndian, data)
}

func (c *cache) Entry(id Id, data interface{}) error {
	if err := c.setOffset(id); err != nil {
		return err
	}

	return binary.Read(c.file, binary.LittleEndian, data)
}

func (c *cache) Close() error {
	return c.file.Close()
}

func (c *cache) setOffset(id Id) error {
	return setOffset(c.file, id, c.chunkSize)
}

func (dc *dynamicCache) SetEntry(id Id, data interface{}) error {
	if err := dc.setOffset(id, data); err != nil {
		return err
	}

	return binary.Write(dc.file, binary.LittleEndian, data)
}

func (dc *dynamicCache) Entry(id Id, data interface{}) error {
	if err := dc.setOffset(id, data); err != nil {
		return err
	}

	return binary.Read(dc.file, binary.LittleEndian, data)
}

func (dc *dynamicCache) Close() error {
	return dc.file.Close()
}

func (dc *dynamicCache) setOffset(id Id, data interface{}) error {
	return setOffset(dc.file, id, int64(binary.Size(data)))
}

func setOffset(file *os.File, id Id, chunkSize int64) error {
	offset := int64(id) * chunkSize
	_, err := file.Seek(offset, io.SeekStart)

	return err
}
