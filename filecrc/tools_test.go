package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	TestZipName  string = "testfiles/Dummy.zip"
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
	err := compressFile(TestZipName, TestPassword, TestFileName, buff.Bytes())

	a.Nil(err)

	// Read the file from the zip and look at the contents
	byteValues, err := uncompressFile(TestZipName, TestFileName, TestPassword)
	a.Nil(err)

	// Convert the resonse to a string
	fileContents := string(byteValues)
	a.Equal(TestContents, fileContents)
}

func TestCRC(t *testing.T) {
	a := assert.New(t)

	info, err := computeFileCRC64("testfiles/a-values.test")

	a.Nil(err)

	fmt.Printf("CRC is %#v\n", info)

	line := buildCRCLine(info)
	fmt.Println(line)

	otherInfo, err := parseCRCLine(line)
	a.Nil(err)

	fmt.Printf("%#v\n", otherInfo)
}

func TestBuildNewName(t *testing.T) {
	testNames := []string{"testfiles/abc.txt", "testfiles/abc1.txt", "testfiles/abc2.txt", "testfiles/abc"}

	for _, name := range testNames {
		os.Create(name)
	}

}
func TestBuildName(t *testing.T) {
	a := assert.New(t)

	testNames := []string{"testfiles/abc.txt", "testfiles/abc1.txt", "testfiles/abc2.txt", "testfiles/abc"}

	for _, name := range testNames {
		os.Create(name)
	}

	newName, err := buildFileName("testfiles/xyz.txt")
	a.Nil(err)
	a.Equal(newName, "testfiles/xyz.txt")

	newName, err = buildFileName("testfiles/abc.txt")
	a.Nil(err)
	a.Equal(newName, "testfiles/abc3.txt")

	newName, err = buildFileName("testfiles/abc")
	a.Nil(err)
	a.Equal(newName, "testfiles/abc1")

	if _, err := os.Stat(testNames[0]); !os.IsNotExist(err) {
		fmt.Printf("The file %s exists\n", testNames[0])
	} else {
		a.Errorf(err, "File exists error")
	}

	missing := "dummy"
	if _, err := os.Stat(missing); os.IsNotExist(err) {
		fmt.Printf("The file %s does not exists\n", missing)
	} else {
		a.Errorf(err, "File does not exist error")
	}
}
