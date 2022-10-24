package main

import (
	"hash/crc64"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dacdb.com/GoCode/filecrc/utils"
)

// computeFileCRC64 Compute the CRC64 for the file specified in the path
// path: Full pathname of the file to compute the crc64 for
// returns CRC64 for the file contents
func computeFileCRC64(path string) (utils.FileInfo, error) {
	// Setup an empty response for errors
	response := utils.FileInfo{}

	// Get the file stat info
	info, err := os.Stat(path)

	if err != nil {
		return response, err
	}

	// Get the date/time
	fs := info.Sys().(*syscall.Win32FileAttributeData)
	response.SetAccessed(time.Unix(0, fs.LastAccessTime.Nanoseconds()))
	response.SetModified(time.Unix(0, fs.LastWriteTime.Nanoseconds()))
	response.SetCreated(time.Unix(0, fs.CreationTime.Nanoseconds()))

	if parmExcludeCRC {
		response.SetCRC(0)
	} else {
		// Open the file
		fileRdr, err := os.Open(path)

		if err != nil {
			return response, err
		}

		// Read the file in
		content := make([]byte, info.Size())

		if _, err := io.ReadFull(fileRdr, content); err != nil {
			return response, err
		}

		// Compute the crc
		response.SetCRC(crc64.Checksum(content, crc64.MakeTable(crc64.ECMA)))

		// Cleanup
		err = fileRdr.Close()

		if err != nil {
			return response, err
		}

		// Reset the times to before reading it
		err = os.Chtimes(path, response.GetAccessed(), response.GetModified())

		if err != nil {
			return response, err
		}

		// Help out the garbage collection
		content = nil
	}

	// Set the name and size
	response.SetName(path)
	response.SetSize(info.Size())

	// Return result
	return response, nil
}

// buildFileName Builds a filename based on the suggested name.  If it exists, a numeric suffix is added until unique
func buildFileName(fileName string, isTemp bool) (string, error) {
	// Cleanup first
	fileName = strings.ReplaceAll(fileName, `\`, "/")

	// Get the containing directory name
	var path string
	slashIndex := strings.LastIndex(fileName, "/")

	if slashIndex >= 0 {
		path = fileName[0 : slashIndex+1]
		fileName = fileName[slashIndex+1:]
	} else {
		path = "./"
	}

	// Get the list of file info for the path
	fileList, err := ioutil.ReadDir(path)

	if err != nil {
		return "", err
	}

	// See if the specified file exists in the list, good if it is
	found := false
	for _, info := range fileList {
		if info.IsDir() {
			continue
		}

		if strings.EqualFold(info.Name(), fileName) {
			found = true
			break
		}
	}

	// If not found, use the filename
	if !found && !isTemp {
		return path + fileName, nil
	}

	// Take the name apart
	var baseName string
	var suffix string
	dotIndex := strings.LastIndex(fileName, ".")

	if dotIndex >= 0 {
		baseName = fileName[0:dotIndex]
		suffix = fileName[dotIndex:]
	} else {
		baseName = fileName
		suffix = ""
	}

	// If using a temp name, modify the base name
	if isTemp {
		baseName = baseName + "-Tmp"
	}

	// Find the highest suffix using a regular expression to isolate a possible suffix

	nameRegEx := regexp.MustCompile("^" + baseName + "([0-9]+)" + suffix + "$")
	maxNumber := 0
	for _, info := range fileList {
		if info.IsDir() {
			continue
		}

		matches := nameRegEx.FindAllStringSubmatch(info.Name(), -1)
		if len(matches) > 0 {
			num, _ := strconv.Atoi(matches[0][1])
			if num > maxNumber {
				maxNumber = num
			}
		}
	}

	// New name
	newName := path + baseName + strconv.Itoa(maxNumber+1) + suffix

	// Return the new name
	return newName, nil
}
