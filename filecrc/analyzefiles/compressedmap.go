package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"io/ioutil"
	"strings"
)

type CompressedMap struct {
	content map[string][]byte
}

func (s *CompressedMap) Init(size int) {
	s.content = make(map[string][]byte, size)
}

func (s *CompressedMap) Put(name, value string) {
	key := strings.ToLower(name)
	s.content[key] = gzipCompress(value)
}

func (s *CompressedMap) Get(name string) (string, bool) {
	key := strings.ToLower(name)

	mapData, found := s.content[key]
	if !found {
		return "", false
	}

	return gzipDecompress(mapData), true
}

func gzipCompress(value string) []byte {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)

	if _, err := gz.Write([]byte(value)); err != nil {
		panic(err)
	}

	if err := gz.Flush(); err != nil {
		panic(err)
	}

	if err := gz.Close(); err != nil {
		panic(err)
	}

	// Return the compressed byte slice
	return compressed.Bytes()
}

func gzipDecompress(compressed []byte) string {
	rdata := bytes.NewReader(compressed)
	r, _ := gzip.NewReader(rdata)
	uncompressed, _ := ioutil.ReadAll(r)
	r.Close()

	// Return the decompressed bytes as a string
	return string(uncompressed)
}

func (s *CompressedMap) PutZ(name string, value string) {
	key := strings.ToLower(name)
	s.content[key] = zlibCompress(value)
}

func (s *CompressedMap) GetZ(name string) (string, bool) {
	key := strings.ToLower(name)

	compressed, found := s.content[key]

	if !found {
		return "", false
	}

	return zlipDecompress(compressed), true
}

func zlibCompress(value string) []byte {
	var compressed bytes.Buffer
	zWtr := zlib.NewWriter(&compressed)

	zWtr.Write([]byte(value))
	zWtr.Close()

	return compressed.Bytes()
}

func zlipDecompress(compressed []byte) string {
	// Uncompress the data into a string
	rdr := bytes.NewReader(compressed)
	zRdr, err := zlib.NewReader(rdr)

	if err != nil {
		panic("Cannot uncompress data")
	}

	var uncompressed bytes.Buffer
	wtr := io.Writer(&uncompressed)

	io.Copy(wtr, zRdr)
	zRdr.Close()

	return uncompressed.String()
}
