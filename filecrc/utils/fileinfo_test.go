package utils

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	a := assert.New(t)

	info := FileInfo{}
	info.name = "rec1"
	info.accessed = time.Now()
	info.modified = time.Now()
	info.created = time.Now()
	info.crc = 12345
	info.size = 10

	str1 := info.BuildCRCLine()
	a.True(len(str1) > 0)

	info.SetAdded()
	info.SetMismatched()

	str2 := info.BuildCRCLine()
	fmt.Fprintf(os.Stdout, "%s\n", str2)

	info.SetSuspicious()
	str3 := info.BuildCRCLine()
	fmt.Fprintf(os.Stdout, "%s\n", str3)

	info2 := FileInfo{}
	info2.ParseCRCLine(str3)
	str4 := info.BuildCRCLine()

	a.Equal(str3, str4)

	info2.ClearFlag()
	a.True(info2.flag == 0x00)
}
