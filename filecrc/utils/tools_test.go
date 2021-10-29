package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	TestZipName  string = "Dummy.zip"
	TestFileName string = "content.txt"
	TestPassword string = "dummydummy"
	TestContents string = "Line 1\nLine 2\n"
)

func TestZip(t *testing.T) {
	a := assert.New(t)

	// Create a byte slice of the string to simulate actual usage
	var buff bytes.Buffer
	buff.WriteString(TestContents)

	// Compress the file first and check
	err := CompressFile(TestZipName, TestPassword, TestFileName, buff.Bytes())

	a.Nil(err)

	// Read the file from the zip and look at the contents
	byteValues, err := UncompressFile(TestZipName, TestFileName, TestPassword)
	a.Nil(err)

	// Convert the resonse to a string
	fileContents := string(byteValues)
	a.Equal(TestContents, fileContents)
}
