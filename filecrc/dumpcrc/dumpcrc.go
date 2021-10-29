package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"dacdb.com/GoCode/filecrc/utils"
	"github.com/pborman/getopt/v2"
)

const (
	flagSuspicious = 's' // Dump suspicious records
	flagAdded      = 'a' // Dump added records
	flagModified   = 'm' // Dump modified records
	flagFilename   = 'f' // Specify the File name in the zip file
	flagPassword   = 'p' // Specify the zip file password
	flagNameRegex  = 'n' // Specify a name regular expression
)

var (
	parmSuspicious  bool   = false // Indicator to dump suspicious records
	parmAdded       bool   = false // Indicator to dump inserted records
	parmModified    bool   = false // Indicator to dump modified records
	parmFilename    string = ""    // File name in the zip file
	parmZipPassword string = ""    // Optional zip file password
	parmInputFile   string = ""    // Input file name
	parmNameRegex   string = ""    // Name search regular expression

	fileData  []byte         = nil // Uncompressed content of the file from the zip
	nameRegex *regexp.Regexp = nil // Compiled name regular expression, if specified

	totalCt      int = 0 // Total number of records processed
	suspiciousCt int = 0 // Number of suspicious records
	insertedCt   int = 0 // Number of records inserted
	modifiedCt   int = 0 // Number of modified records
	unchangedCt  int = 0 // Number of unchanged records
	selectedCt   int = 0 // Number of records selected
)

func main() {
	// Need some args
	if err := getParms(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		usage()
		os.Exit(2)
	}

	// Initialize
	if err := initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(2)
	}

	// Process
	rc := process()

	os.Exit(rc)
}

// getParms Parse the command line arguments
func getParms(args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("you must specify configuration parameters")
	}

	// Setup for processing
	flagSet := getopt.New()

	// Setup specs
	flagSet.Flag(&parmSuspicious, flagSuspicious, "Dump suspicious files")
	flagSet.Flag(&parmAdded, flagAdded, "Dump added files")
	flagSet.Flag(&parmModified, flagModified, "Dump modified files")
	flagSet.Flag(&parmFilename, flagFilename, "Specify the file name in the zip file")
	flagSet.Flag(&parmZipPassword, flagPassword, "Specify the zip file password (optional)")
	flagSet.Flag(&parmNameRegex, flagNameRegex, "Regular expression for searching for a name")

	// Parse the arguments
	err := flagSet.Getopt(args, nil)

	if err != nil {
		return err
	}

	// Get the input file name
	posArgs := flagSet.Args()

	if len(posArgs) != 1 {
		return fmt.Errorf("you must specify an input file name")
	}

	parmInputFile = posArgs[0]

	// Some checking
	if len(parmNameRegex) > 0 {
		if parmAdded || parmModified || parmSuspicious {
			return fmt.Errorf("the name search is mutually exclusive with other parameters")
		}

		// Compile the name regular expression if specified
		if nameRegex, err = regexp.Compile(parmNameRegex); err != nil {
			return err
		}
	}

	// Return success
	return nil
}

// usage Display program usage
func usage() {
	fmt.Fprintf(os.Stderr, "Dump records from the CRC zip file based on types\n")
	fmt.Fprintf(os.Stderr, "Usage: [[-%c] [-%c] [-%c]]|[-%c nameRegExp] [-%c zipPassword] [-%c filename] zipFileName\n",
		flagSuspicious, flagAdded, flagModified, flagNameRegex, flagPassword, flagFilename)
	fmt.Fprintf(os.Stderr, " %c: Dump info for suspicious files\n", flagSuspicious)
	fmt.Fprintf(os.Stderr, " %c: Dump info for added files\n", flagAdded)
	fmt.Fprintf(os.Stderr, " %c: Dump info for modified filed\n", flagModified)
	fmt.Fprintf(os.Stderr, " %c: Dump info for file names matching the specified regular expression\n", flagNameRegex)
	fmt.Fprintf(os.Stderr, "     Note: This parameter is mutually exclusive with the other selection parameters\n")
	fmt.Fprintf(os.Stderr, " %c fileName: Name of the file to dump in the zip (default: first file found)\n", flagFilename)
	fmt.Fprintf(os.Stderr, " %c password: Use the following password for the zip file\n", flagPassword)
	fmt.Fprintf(os.Stderr, " Note: If no options are given, then a summary count is produced\n")
	fmt.Fprintf(os.Stderr, "\n")
}

// initialize Open the files and get started
func initialize() error {
	// Uncompress the zip file
	var err error
	if fileData, err = utils.UncompressFile(parmInputFile, parmFilename, parmZipPassword); err != nil {
		return err
	}

	// Return success
	return nil
}

// process Read each record from the file and see if you need to display
func process() int {
	// Process each record in
	// Create a reader over the data
	reader := bytes.NewReader(fileData)

	// RAllocate a buffer for long lines and setup a scanner
	scanBuff := make([]byte, 500000)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(scanBuff, 500000)

	// Read each line of the buffer
	lineNo := 0
	for scanner.Scan() {
		// Process each line
		lineNo++
		currentLine := scanner.Text()

		fileInfo := utils.FileInfo{}
		err := fileInfo.ParseCRCLine(currentLine)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return 2
		}

		// Increment the counts
		totalCt++

		// Select the requested records
		if nameRegex == nil {
			// Use selection criteria
			if fileInfo.GetFlag() != 0 {
				if fileInfo.IsSuspicious() {
					suspiciousCt++
				}

				if fileInfo.IsAdded() {
					insertedCt++
				}

				if fileInfo.IsMismatched() {
					modifiedCt++
				}

				// Check the record
				if (parmSuspicious && fileInfo.IsSuspicious()) || (parmAdded && fileInfo.IsAdded()) || (parmModified && fileInfo.IsMismatched()) {
					fileInfo.Display(os.Stdout)
					selectedCt++
				}
			} else {
				// No flags indicated so just accumulate the count
				unchangedCt++
			}
		} else {
			// A regular expression for the name was specified, us it

			// Normalize slashes and break down the parts of the name
			name := strings.ReplaceAll(fileInfo.GetName(), "\\", "/")
			names := strings.Split(name, "/")

			// Compare against each component of the name
			for _, component := range names {
				if nameRegex.MatchString(component) {
					fileInfo.Display(os.Stdout)
					selectedCt++
					// No need to continue if you got one match
					break
				}
			}
		}
	}

	// Print stats
	printStats()

	return 0
}

func printStats() {
	if selectedCt > 0 {
		// Print a blank line if ny records displayed, for spacing
		fmt.Fprintf(os.Stdout, "\n")
	}

	// Print the rest of the stats
	fmt.Fprintf(os.Stdout, "Total records read:      %s\n", utils.NiceInt(totalCt))
	fmt.Fprintf(os.Stdout, "Total records selected:  %s\n", utils.NiceInt(selectedCt))
	fmt.Fprintf(os.Stdout, "Suspicious records read: %s\n", utils.NiceInt(suspiciousCt))
	fmt.Fprintf(os.Stdout, "Inserted  records read:  %s\n", utils.NiceInt(insertedCt))
	fmt.Fprintf(os.Stdout, "Modified records read:   %s\n", utils.NiceInt(modifiedCt))
	fmt.Fprintf(os.Stdout, "Unchanged records read:  %s\n", utils.NiceInt(unchangedCt))
}
