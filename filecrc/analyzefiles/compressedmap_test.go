package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	str10  = "0123456789"
	str20  = "01234567890123456789"
	str40  = "0123456789012345678901234567890123456789"
	str80  = "01234567890123456789012345678901234567890123456789012345678901234567890123456789"
	str120 = "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
)

func TestZlibCompression(t *testing.T) {
	fmt.Printf("Testing zlib compression\n")
	doZlibCompress(str10)
	doZlibCompress(str20)
	doZlibCompress(str40)
	doZlibCompress(str80)
	doZlibCompress(str120)
}

func doZlibCompress(value string) {
	compressed := zlibCompress(value)
	fmt.Printf("In: %d, out:%d\n", len(value), len(compressed))
}

func TestGZipCompression(t *testing.T) {
	fmt.Printf("Testing gzip compression\n")
	doGZipCompress(str10)
	doGZipCompress(str20)
	doGZipCompress(str40)
	doGZipCompress(str80)
	doGZipCompress(str120)
}

func doGZipCompress(value string) {
	compressed := gzipCompress(value)
	fmt.Printf("In: %d, out:%d\n", len(value), len(compressed))
}

func TestGZipMap(t *testing.T) {
	a := assert.New(t)

	var stats CompressedMap
	stats.Init(10)
	stats.Put("key1", str120)

	text, status := stats.Get("key1")

	a.True(status)
	a.Equal(text, str120)
}
func TestZlibMap(t *testing.T) {
	a := assert.New(t)

	var stats CompressedMap
	stats.Init(10)
	stats.PutZ("key1", str120)

	text, status := stats.GetZ("key1")

	a.True(status)
	a.Equal(text, str120)
}
