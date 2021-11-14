package main

import (
	"fmt"
	"hash/crc64"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pborman/getopt/v2"
)

const (
	FlagModTime    = 'm' // Set the modification dat/time
	FlagAccessTime = 'a' // Set the last accessed date/time
	flagExcludeCRC = 'x' // Exclude computing the CRC
)

var (
	parmModSpec    string = "" // Modification spec date/time to set
	parmAccessSpec string = "" // Last accessed spec date/time to set
	parmFileName   string = "" // Filename to manipulate

	parmAccessTime time.Time // Parsed access time
	parmModTime    time.Time // Parsed last accessed time

	parmExcludeCRC bool = false // Compute the CRC

	runtimeGetInfoOnly bool = true // Flag to indicate getting the info only
)

type FileStats struct {
	accessed string // Last accessed date and time
	modified string // Last modified date and time
	created  string // Created date and time
	size     int64  // File size
	crc64    uint64 // File CRC
}

func main() {
	if status := parseParms(); !status {
		usage(os.Stderr)
		os.Exit(2)
	}

	var successFull = true
	if runtimeGetInfoOnly {
		// Get the file info with the CRC
		stats := getFileInfo(parmFileName, !parmExcludeCRC)

		// Print the stats
		if stats != nil {
			printStats(stats)
		} else {
			successFull = false
		}
	} else {
		// Set the file info redisplay the stats
		successFull = setFileInfo(parmFileName)
		if successFull {
			// Get the stats without the CRC
			stats := getFileInfo(parmFileName, false)

			if stats != nil {
				printStats(stats)
			} else {
				successFull = false
			}
		}
	}

	// Unsuccessfull exit
	if !successFull {
		os.Exit(2)
	}

	// Normal exit
	os.Exit(0)
}

// parseParms Parse the command line parameters
func parseParms() bool {
	// Setup for processing
	flagSet := getopt.New()

	// Setup specs
	flagSet.Flag(&parmModSpec, FlagModTime, "Set the file modification time")
	//flagSet.Lookup(FlagModTime).SetOptional()

	flagSet.Flag(&parmAccessSpec, FlagAccessTime, "Set the file last accessed time")
	//flagSet.Lookup(FlagAccessTime).SetOptional()

	flagSet.Flag(&parmExcludeCRC, flagExcludeCRC, "Set the file last accessed time")

	// Parse the flags
	err := flagSet.Getopt(os.Args, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing command line: %s\n", err)
		return false
	}

	// Validate everything in one pass
	successFull := true

	// Parse the specs if provided
	if len(parmAccessSpec) > 0 {
		var err error = nil

		if strings.ToLower(parmAccessSpec) == "now" {
			parmAccessTime = time.Now()
		} else {
			parmAccessTime, err = time.Parse(time.RFC3339Nano, parmAccessSpec)
		}

		if err != nil {
			successFull = false
			fmt.Fprintf(os.Stderr, "Error parsing last access time '%s': %s\n", parmAccessSpec, err)
		}
	}

	if len(parmModSpec) > 0 {
		var err error = nil

		if strings.ToLower(parmModSpec) == "now" {
			parmModTime = time.Now()
		} else {
			parmModTime, err = time.Parse(time.RFC3339Nano, parmModSpec)
		}

		if err != nil {
			successFull = false
			fmt.Fprintf(os.Stderr, "Error parsing modified time '%s': %s\n", parmModSpec, err)
		}
	}

	// Get the file name to process
	args := flagSet.Args()

	if len(args) != 1 {
		successFull = false
		fmt.Fprintf(os.Stderr, "You must specify a file name to process\n")
		return false
	}

	// Save the argument
	parmFileName = args[0]

	// See if you are doing any mods
	if flagSet.IsSet(FlagAccessTime) || flagSet.IsSet(FlagModTime) {
		runtimeGetInfoOnly = false
	}

	return successFull
}

// usage Display program usage
func usage(out *os.File) {
	fmt.Fprintf(out, "Usage: [-%c accessSpec] [-%c modifiedSpec] [-%c] filename\n", FlagAccessTime, FlagModTime, flagExcludeCRC)
	fmt.Fprintf(out, "When run without any options, the info for the file is displayed\n")
	fmt.Fprintf(out, "  accessSpec:   A date/time specification for setting the last accessed date/time\n")
	fmt.Fprintf(out, "                a spec of 'now' will use the current date/time\n")
	fmt.Fprintf(out, "  modifiedSpec: A date/timespecification for setting the modified date/time\n")
	fmt.Fprintf(out, "                a spec of 'now' will use the current date/time\n")
	fmt.Fprintf(out, "  x: exclude computing CRC\n")
	fmt.Fprintf(out, "Note: All specifications follow RFC3339Nano (%s)\n", time.RFC3339Nano)
	fmt.Fprintf(out, "\n")
}

// getFileInfo Get the access, modified, create times and crc for the specified file
func getFileInfo(fileName string, buildCRC bool) *FileStats {
	// Get the last accseed date/time before reading in the file for generating the crc
	info, err := os.Stat(fileName)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stating file %s: %s\n", fileName, err)
		return nil
	}

	fs := info.Sys().(*syscall.Win32FileAttributeData)
	lastAccessed := time.Unix(0, fs.LastAccessTime.Nanoseconds())
	lastModified := time.Unix(0, fs.LastWriteTime.Nanoseconds())

	// Build the CRC if requested

	var crc uint64
	if buildCRC {
		// Open the file and read it to build the CRC
		fileRdr, err := os.Open(fileName)

		if err != nil {
			return nil
		}

		content := make([]byte, info.Size())

		// Read the file in

		_, err = io.ReadFull(fileRdr, content)

		// Close the file and reset the last accessed date/time
		fileRdr.Close()
		os.Chtimes(fileName, lastAccessed, lastModified)

		// Return if read error
		if err != nil {
			return nil
		}

		crc = crc64.Checksum(content, crc64.MakeTable(crc64.ECMA))

		// Help garbage collection
		content = nil
	} else {
		crc = 0
	}

	stats := FileStats{}
	stats.accessed = lastAccessed.Format(time.RFC3339Nano)
	stats.modified = lastModified.Format(time.RFC3339Nano)
	stats.created = time.Unix(0, fs.CreationTime.Nanoseconds()).Format(time.RFC3339Nano)
	stats.size = info.Size()
	stats.crc64 = crc

	return &stats
}

// Set the file info from the command line arguments
func setFileInfo(fileName string) bool {
	// Get the current stats since you can set either access of modified
	info, err := os.Stat(fileName)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stating file %s: %s\n", fileName, err)
		return false
	}

	fs := info.Sys().(*syscall.Win32FileAttributeData)
	lastAccessed := time.Unix(0, fs.LastAccessTime.Nanoseconds())
	lastModified := time.Unix(0, fs.LastWriteTime.Nanoseconds())

	if len(parmAccessSpec) == 0 {
		// Use files current value if not specified
		parmAccessTime = lastAccessed
	}

	if len(parmModSpec) == 0 {
		// Use files current value if not specified
		parmModTime = lastModified
	}

	// Change the times
	err = os.Chtimes(fileName, parmAccessTime, parmModTime)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting file '%s' times: %s\n", fileName, err)
		return false
	}

	return true
}

// printStats Print the file stat info
func printStats(stats *FileStats) {
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "Information for file: %s\n", parmFileName)
	fmt.Fprintf(os.Stdout, "Created:  %s\n", stats.created)
	fmt.Fprintf(os.Stdout, "Modified: %s\n", stats.modified)
	fmt.Fprintf(os.Stdout, "Accessed: %s\n", stats.accessed)
	fmt.Fprintf(os.Stdout, "Size:     %s\n", humanize.Comma(int64(stats.size)))

	if stats.crc64 != 0 {
		fmt.Fprintf(os.Stdout, "CRC64:    %d\n", stats.crc64)
	}
}
