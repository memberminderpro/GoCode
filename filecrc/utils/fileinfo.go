package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Constants

const (
	FieldSep            = "|"  // Output file field separator
	flagSuspicious byte = 0x01 // Flag to indicate the record is suspicious
	flagMismatched byte = 0x02 // Flag to indicate the record has changed
	flagInserted   byte = 0x04 // Flag to indicate the record is new

	prefixSize     = 3   // Size of the file status prefix excluding the colon
	prefixChar     = ':' // Prefix separator
	codeMissing    = "-" // Code to inidicate the field is not set
	codeSusicious  = "S" // Code indicating the record is suspicious
	codeMismatched = "M" // Code to indicate the record is modified (mismatched)
	codeInserted   = "N" // Code to indicate the record is new
)

// File Info
type FileInfo struct {
	name     string    // Name of the file
	modified time.Time // Last modified date
	accessed time.Time // Last accessed time
	created  time.Time // Created time
	size     int64     // File size
	crc      uint64    // Computed CRC
	flag     byte      // Indicator flags
}

func (info *FileInfo) GetName() string {
	return info.name
}
func (info *FileInfo) GetModified() time.Time {
	return info.modified
}

func (info *FileInfo) GetAccessed() time.Time {
	return info.accessed
}

func (info *FileInfo) GetCreated() time.Time {
	return info.created
}

func (info *FileInfo) GetSize() int64 {
	return info.size
}

func (info *FileInfo) GetCRC() uint64 {
	return info.crc
}

func (info *FileInfo) GetFlag() byte {
	return info.flag
}

func (info *FileInfo) SetName(value string) {
	info.name = value
}

func (info *FileInfo) SetAccessed(value time.Time) {
	info.accessed = value
}

func (info *FileInfo) SetModified(value time.Time) {
	info.modified = value
}

func (info *FileInfo) SetCreated(value time.Time) {
	info.created = value
}

func (info *FileInfo) SetCRC(value uint64) {
	info.crc = value
}

func (info *FileInfo) SetSize(value int64) {
	info.size = value
}

func (info *FileInfo) SetFlag(value byte) {
	info.flag = value
}

func (info *FileInfo) IsEqual(other FileInfo) bool {
	return info.TimesEqual(other) && info.size == other.size && info.crc == other.crc
}

func (info *FileInfo) TimesEqual(other FileInfo) bool {
	return info.created == other.created && info.accessed == other.accessed && info.modified == other.modified
}

// buildCRCLine  Construct a CRC summary line with a separator
func (info *FileInfo) BuildCRCLine() string {
	return fmt.Sprintf("%s%c%s%s%s%s%s%s%s%s%d%s%d\n",
		info.GetStatus(), prefixChar,
		info.name,
		FieldSep, info.created.Format(time.RFC3339Nano),
		FieldSep, info.accessed.Format(time.RFC3339Nano),
		FieldSep, info.modified.Format(time.RFC3339Nano),
		FieldSep, info.size,
		FieldSep, info.crc)
}

func (info *FileInfo) Display(log *os.File) {
	fmt.Fprintf(log, "File name: %s\nFlags: '%s', Size: %s, CRC: %d\nCreated: %s, Modified: %s, Accessed: %s\n",
		info.name, info.GetStatus(), NiceInt64(info.size), info.crc,
		info.created.Format(time.RFC3339Nano), info.modified.Format(time.RFC3339Nano), info.accessed.Format(time.RFC3339Nano))
}

// parseCRCLine Parse a line of file status into a CRCInfo
func (info *FileInfo) ParseCRCLine(line string) error {
	// Remove trailing newline if there
	line = strings.TrimSuffix(line, "\n")

	// Split the string first
	parts := strings.Split(line[prefixSize+1:], FieldSep)

	if len(parts) != 6 {
		return fmt.Errorf("the line '%s' is invalid", line)
	}

	// Build the info
	var err error = nil

	// Get the file name
	part := 0
	info.name = parts[part]

	// Parse the created date/time
	part++
	if info.created, err = time.Parse(time.RFC3339Nano, parts[part]); err != nil {
		return err
	}

	// Parse the accessed date/time
	part++
	if info.accessed, err = time.Parse(time.RFC3339Nano, parts[part]); err != nil {
		return err
	}

	// Parse the modified date/time
	part++
	if info.modified, err = time.Parse(time.RFC3339Nano, parts[part]); err != nil {
		return err
	}

	// Parse the file size
	part++
	if info.size, err = strconv.ParseInt(parts[part], 10, 64); err != nil {
		return err
	}

	// Parse the CRC64
	part++
	if info.crc, err = strconv.ParseUint(parts[part], 10, 64); err != nil {
		return err
	}

	// Set the flag
	info.BuildFlag(line[0:prefixSize])

	// Success
	return nil
}

func (info *FileInfo) SetMismatched() {
	info.flag |= flagMismatched
}

func (info *FileInfo) SetSuspicious() {
	info.flag |= flagSuspicious
}

func (info *FileInfo) SetAdded() {
	info.flag |= flagInserted
}

func (info *FileInfo) IsSuspicious() bool {
	return info.flag&flagSuspicious != 0
}

func (info *FileInfo) IsMismatched() bool {
	return info.flag&flagMismatched != 0
}

func (info *FileInfo) IsAdded() bool {
	return info.flag&flagInserted != 0
}

func (info *FileInfo) ClearFlag() {
	info.flag = 0x00
}

func (info *FileInfo) GetStatus() string {
	status := ""

	if info.flag&flagSuspicious > 0 {
		status += codeSusicious
	} else {
		status += codeMissing
	}

	if info.flag&flagMismatched > 0 {
		status += codeMismatched
	} else {
		status += codeMissing
	}

	if info.flag&flagInserted > 0 {
		status += codeInserted
	} else {
		status += codeMissing
	}

	// Return the status
	return status
}

func (info *FileInfo) BuildFlag(prefix string) {
	info.flag = 0

	if string(prefix[0]) != codeMissing {
		info.flag |= flagSuspicious
	}

	if string(prefix[1]) != codeMissing {
		info.flag |= flagMismatched
	}

	if string(prefix[2]) != codeMissing {
		info.flag |= flagInserted
	}
}
