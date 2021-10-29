package main

import (
	"fmt"
	"os"
	"testing"

	"dacdb.com/GoCode/filecrc/utils"
	"github.com/stretchr/testify/assert"
)

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

	newName, err := buildFileName("testfiles/xyz.txt", false)
	a.Nil(err)
	a.Equal(newName, "testfiles/xyz.txt")

	newName, err = buildFileName("testfiles/abc.txt", false)
	a.Nil(err)
	a.Equal(newName, "testfiles/abc3.txt")

	newName, err = buildFileName("testfiles/abc", false)
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

func TestCRC(t *testing.T) {
	a := assert.New(t)

	info, err := computeFileCRC64("testfiles/a-values.test")

	a.Nil(err)

	fmt.Printf("CRC is %#v\n", info)

	line := info.BuildCRCLine()
	fmt.Println(line)

	otherInfo := utils.FileInfo{}
	err = otherInfo.ParseCRCLine(line)
	a.Nil(err)

	fmt.Printf("%#v\n", otherInfo)
}
