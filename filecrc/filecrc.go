package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Runtime parameters from the config
var rootDir string                                // Starting directory
var rootDirSize int                               // Size of the root directory used for making slices to get end dir
var zipFileName string                            // Zip file name
var outputFileName string                         // Name of the output file in the zip file
var emailCredentials credentials                  // Credentials for accessing email
var zipPassword string                            // The password for the encrypted zip file
var emailFrom string                              // Email from email address
var emailToList []string = make([]string, 0)      // To distribution list
var emailCCList []string = make([]string, 0)      // Email CC distribution list
var emailSubject string                           // Email subject
var emailAttachments []string = make([]string, 0) // Collection of logical file names to attach
var logFileName string                            // Name for logging (defaults to stderr)

// Variables used at runtime
var emailFlag bool = true                                          // Flag to indicate tha to send emails at the end
var buildOnlyFlag bool = false                                     // Flag to indicate only building the zip file
var logWriter *os.File = os.Stdout                                 // Default log output
var newZipFileName string                                          // Output zip file name
var excludes map[string]interface{} = make(map[string]interface{}) // Map of directories to exclude
var mismatchedEntries int = 0                                      // Number of entries that did not match
var newEntries int = 0                                             // Number of newly added entries
var fileMap map[string]crcInfo = make(map[string]crcInfo, 10000)   // Collection of CRC info for each file

func main() {
	// Initialize and check status
	parseErrs := getParms()

	if parseErrs != nil {
		fmt.Fprintln(os.Stderr, parseErrs)
		usage()
		os.Exit(2)
	}

	// CLose the log file if not stdout
	if logWriter != os.Stdout {
		defer logWriter.Close()
	}

	// Setup
	var err error = nil

	// Do some initialization
	err = initialize()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization failed: %s\n", err)
		os.Exit(2)
	}

	// Process everything
	err = process()

	if err != nil {
		fmt.Fprintln(logWriter, err)
	}

	// Display some stats before the email so they're in the logfile
	fmt.Fprintf(logWriter, "Processing completed\n")
	fmt.Fprintf(logWriter, "Number of mismatched entries: %d\n", mismatchedEntries)
	fmt.Fprintf(logWriter, "Number of entries added:      %d\n", newEntries)

	// Send emails if requested
	status := err
	err = processEmail(status)

	if err != nil {
		fmt.Fprintf(logWriter, "Error processing\n")
	}

	if err != nil {
		os.Exit(2)
	}

	os.Exit(0)
}

// initialize Do some initialization
func initialize() (err error) {
	// try to capture all errors in a pass
	success := true
	// Create a new zip output file name
	newZipFileName, err = buildFileName(zipFileName)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building output zip name from '%s': %s\n", zipFileName, err)
		return err
	}

	// Swap zip file names, if the zip file name exists
	if _, err := os.Stat(zipFileName); !os.IsNotExist(err) {
		fmt.Fprintf(logWriter, "Existing zip file '%s' is now %s\n", zipFileName, newZipFileName)

		// Do the renaming
		err = os.Rename(zipFileName, newZipFileName)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error renaming '%s' to '%s': %s\n", zipFileName, newZipFileName, err)
			success = false
		}

		// Zip file exists to it's not the first run
		buildOnlyFlag = false
	} else {
		// Zip file does not exist, so it's the first run
		buildOnlyFlag = true
	}

	// Open the log file, if specified
	if len(logFileName) > 0 {
		logWriter, err = os.Create(logFileName)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file '%s': %s\n", logFileName, err)
			success = false
		}
	}

	// Check status
	if !success {
		fmt.Fprintf(os.Stderr, "Initializtion errors we encountered, processing terminated")
		return fmt.Errorf("Initialization error")
	}

	// Success
	return nil
}

// process Do the mainline processing
func process() (err error) {
	// Load the file if not the first run
	if !buildOnlyFlag {
		// Indicate run type
		fmt.Fprintf(logWriter, "Processing root dir against file %s\n", newZipFileName)

		// Get the file
		byteValues, err := uncompressFile(newZipFileName, outputFileName, zipPassword)

		if err != nil {
			fmt.Fprintf(logWriter, "Error uncompressing the previous zip file %s\n", newZipFileName)
			return err
		}

		err = loadFile(byteValues)

		if err != nil {
			fmt.Fprintf(logWriter, "Error loading the file '%s' zip %s\n", outputFileName, newZipFileName)
			return err
		}
	} else {
		fmt.Fprintf(logWriter, "Loading the first run from the root dir\n")
	}

	// Walk the tree
	err = filepath.Walk(rootDir, walkTree)

	if err != nil {
		fmt.Fprintf(logWriter, "Error initializing directory processing for %s\n", rootDir)
		return err
	}

	// Save the file
	err = saveFile()

	// Success
	return err
}

// Walk the directory tree and oricess all files (not directories or links)
func walkTree(path string, info os.FileInfo, err error) error {
	// Clean up the slashes
	path = strings.ReplaceAll(path, `\`, "/")

	// Only process file names and not directories
	if info.IsDir() {
		// Check if you need toskip this entire directory
		if len(path) > rootDirSize {
			subDir := strings.ToLower(path[rootDirSize:])
			if _, found := excludes[subDir]; found {
				fmt.Fprintf(logWriter, "Skipping direcory: %s\n", path)
				return filepath.SkipDir
			}
		}

		// Skip building a crc on just a directory name
		return nil
	}

	// Compute the CRC64 for the specified file
	data, err := computeFileCRC64(path)

	if err != nil {
		fmt.Fprintf(logWriter, "Error processing file '%s': %s\n", path, err)
		return err
	}

	// If not only building, then lookup the info and compare
	keyName := strings.ToLower(data.name)

	if !buildOnlyFlag {
		fileData, found := fileMap[keyName]

		// if it's found, then do the compare
		if found {
			sameFlag := true
			// Check the crc's are the same
			if data.crc != fileData.crc {
				fmt.Fprintf(logWriter, "File %s crc is different old(%d) new (%d)\n", path, fileData.crc, data.crc)
				sameFlag = false
			}

			// Check the file sizes
			if data.size != fileData.size {
				fmt.Fprintf(logWriter, "File %s size is different old(%d) new (%d)\n", path, fileData.size, data.size)
				sameFlag = false
			}

			// Check the modified dates as strings because of locale differences
			timeOld := fileData.modified.Format(DateTimeLayout)
			timeNew := data.modified.Format(DateTimeLayout)
			if timeOld != timeNew {
				fmt.Fprintf(logWriter, "File %s modified date is different old(%s) new (%s)\n", path,
					timeOld, timeNew)
				sameFlag = false
			}

			if !sameFlag {
				mismatchedEntries++
			}
		} else {
			// Not in the original file so log and add it
			fmt.Fprintf(logWriter, "The file '%s' is new and has been added to the output\n", path)
			newEntries++
		}
	} else {
		// Increment the count
		newEntries++
	}

	// Replace or add the data
	fileMap[keyName] = data

	// Return the result
	return nil
}

// loadFile Load the existing file
func loadFile(fileData []byte) error {
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

		crcInfo, err := parseCRCLine(currentLine)

		if err != nil {
			return err
		}

		// Add the entry to the map
		fileMap[strings.ToLower(crcInfo.name)] = crcInfo
	}

	// Return success
	return nil
}

// saveFile Save the intrnal map to a file by creating a file in an encrypted zip
func saveFile() error {
	// Create a buffer and write each entry from the map
	var buff bytes.Buffer

	for _, entry := range fileMap {
		// Create the line for the file and write it into te buffer
		line := buildCRCLine(entry)
		buff.WriteString(line)
	}

	// Build the compressed zip
	err := compressFile(zipFileName, zipPassword, outputFileName, buff.Bytes())

	if err != nil {
		fmt.Fprintf(logWriter, "Error building the compressed zip %s: %s\n", zipFileName, err)
		return err
	}

	// Return success
	return nil
}

func processEmail(err error) error {
	// Make sure we're supposed send the email
	if !emailFlag {
		return nil
	}

	// Build list of file attachments
	attachFiles := make([]string, 0)

	for _, attachType := range emailAttachments {
		switch attachType {
		case AttachLogName:
			attachFiles = append(attachFiles, logFileName)
		case AttachZipName:
			attachFiles = append(attachFiles, zipFileName)
		default:
			fmt.Fprintf(logWriter, "Invalid attachment type for sending an email, %s\n", attachType)
		}
	}

	// setup the message
	var message string
	if err != nil {
		message = "Processing completed <b>in error</b><p>"
	} else {
		message = "Processing completed <b>without errors</b><p>"
	}

	// List attached files
	for _, files := range attachFiles {
		message += "File " + files + " attached<p>"
	}

	// Send the email
	err = sendEmail(emailFrom, emailToList, emailCCList, emailSubject, message, attachFiles, emailCredentials)

	if err != nil {
		fmt.Fprintf(logWriter, "An error occurred sending the email: %s\n", err)
		return err
	}

	// Success
	return nil
}