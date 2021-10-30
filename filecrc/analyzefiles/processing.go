package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"dacdb.com/GoCode/filecrc/utils"
	"github.com/karrick/godirwalk"
)

const (
	OneMB          = 1024 * 1024
	MethodFilepath = "filepath"
	MethodGoDir    = "godirwalk"
)

// Runtime parameters from the config
var (
	RoorDir          string                                // Global variable for te recursive tree walking
	rootDirs         []string          = make([]string, 0) // Starting directories
	excludes         []string          = make([]string, 0) // List of files to exclude
	inputZipFile     string                                // Zip file name
	outputFileName   string                                // Name of the output file in the zip file
	emailCredentials utils.Credentials                     // Credentials for accessing email
	zipPassword      string                                // The password for the encrypted zip file
	emailFrom        string                                // Email from email address
	emailToList      []string          = make([]string, 0) // To distribution list
	emailCCList      []string          = make([]string, 0) // Email CC distribution list
	emailSubject     string                                // Email subject
	emailAttachments []string          = make([]string, 0) // Collection of logical file names to attach
	logFileName      string                                // Name for logging (defaults to stderr)
	totalFiles       int               = 0                 // Total nuber of files processed
	printStats       bool              = false             // Flag to display runtime stats
	walkFilepath     bool              = false             // Method for walking the directories
)

// Variables used at runtime
var (
	emailFlag         bool                      = true                                    // Flag to indicate tha to send emails at the end
	compareFields     bool                      = false                                   // Flag to indicate that you can compare fields
	regexExcludes     []*regexp.Regexp          = make([]*regexp.Regexp, 0)               // Array of compiled exclude patterns
	logWriter         *os.File                  = os.Stdout                               // Default log output
	outputZipFile     string                    = ""                                      // Output zip file name
	mismatchedEntries int                       = 0                                       // Number of entries that did not match
	newEntries        int                       = 0                                       // Number of newly added entries
	unchangedEntries  int                       = 0                                       // Number of entries that are the same
	deletedCt         int                       = 0                                       // Number of deleted entries (calculated)
	fileMap           map[string]utils.FileInfo = make(map[string]utils.FileInfo, 100000) // Collection of CRC info for each file
	totalFileSize     int64                     = 0                                       // Total size of all files read
	maxFileSize       int64                     = 0                                       // Maximum file size read
)

// initialize Do some initialization
func run(args []string) int {
	startTime := time.Now()

	// Initialize and check status
	parseErrs := getParms(args)

	if parseErrs != nil {
		fmt.Fprintln(os.Stderr, parseErrs)
		usage()
		return 2
	}

	// Check if only validating the parms
	if parmVerifyConfig {
		fmt.Fprintln(os.Stdout, "The configuration file is correct")

		// All done if not verifying exclusions
		if parmVerifyExclude {
			fmt.Fprintln(os.Stdout, "Verifying exclusion definitions only")
		} else {
			return 0
		}
	}

	// Close the log file if not stdout
	if logWriter != os.Stdout {
		defer logWriter.Close()
	}

	// Setup
	var err error = nil

	// Do some initialization
	err = initialize()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization failed: %s\n", err)
		return 2
	}

	// Display some basic info
	if len(outputFileName) == 0 {
		fmt.Fprintf(logWriter, "The output file name was not spcified, %s us being used\n", DefaultOutputFileName)
	}

	if len(zipPassword) == 0 {
		fmt.Fprintf(logWriter, "No zip encryption password supplied, the file will not be encrypted\n")
	}

	// Process everything
	err = process()

	if err != nil {
		fmt.Fprintln(logWriter, err)
	}

	// Compute number of files deleted
	deletedCt = len(fileMap) - newEntries - unchangedEntries - suspiciousCt - mismatchedEntries

	// Display some stats before the email so they're in the logfile
	fmt.Fprintf(logWriter, "Processing completed\n")
	fmt.Fprintf(logWriter, "Total files processed:        %s\n", utils.NiceInt(totalFiles))
	fmt.Fprintf(logWriter, "Number of suspicious files:   %s\n", utils.NiceInt(suspiciousCt))
	fmt.Fprintf(logWriter, "Number of modified files:     %s\n", utils.NiceInt(mismatchedEntries))
	fmt.Fprintf(logWriter, "Number of files added:        %s\n", utils.NiceInt(newEntries))
	fmt.Fprintf(logWriter, "Number of files deleted:      %s\n", utils.NiceInt(deletedCt))

	// Exclude stats if no CRC
	if !parmExcludeCRC {
		fmt.Fprintf(logWriter, "Total size of files read:     %s\n", utils.NiceInt64(totalFileSize))
		fmt.Fprintf(logWriter, "Maximum file size read:       %s\n", utils.NiceInt64(maxFileSize))
	} else {
		fmt.Fprintf(logWriter, "Total size of files read:     %s\n", "No files were read")
		fmt.Fprintf(logWriter, "Maximum file size read:       %s\n", "No files were read")
	}

	// Display system memory stats if requested
	if printStats {
		PrintMemoryStats("Job end")
	}

	// Send emails if requested
	status := err
	err = processEmail(status)

	if err != nil {
		fmt.Fprintf(logWriter, "Error processing email\n")
		return 2
	}

	// Get stop time
	stopTime := time.Now()
	runTime := stopTime.Sub(startTime)

	fmt.Fprintf(logWriter, "Processing completed in %5.2f minutes\n", runTime.Minutes())

	return 0
}

func initialize() (err error) {
	// try to capture all errors in a pass
	success := true

	// Open the log file, if specified (if verifying excludes, leave it as stdout)
	if len(logFileName) > 0 && !parmVerifyExclude {
		logWriter, err = os.Create(logFileName)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log file '%s': %s\n", logFileName, err)
			success = false
		}
	}

	// Setup file names for general processing

	if parmVerifyExclude {
		// Do nothing
	} else if parmAnalyzeOnly {
		// Use the base name for input if specified
		if len(parmBaseName) > 0 {
			inputZipFile = parmBaseName
		}

		// See if the input file exists
		if _, err := os.Stat(inputZipFile); !os.IsNotExist(err) {
			compareFields = true
		} else {
			compareFields = false
		}
	} else {
		// Build a temp name if comparing to a specific base name
		if len(parmBaseName) == 0 {
			// Create a new zip output file name
			newZipFileName, err := buildFileName(inputZipFile, false)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error building output zip name from '%s': %s\n", inputZipFile, err)
				return err
			}

			// Swap zip file names, if the zip file name exists
			if _, err := os.Stat(inputZipFile); !os.IsNotExist(err) {
				fmt.Fprintf(logWriter, "Existing zip file '%s' is now %s\n", inputZipFile, newZipFileName)

				// Do the renaming
				err = os.Rename(inputZipFile, newZipFileName)

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error renaming '%s' to '%s': %s\n", inputZipFile, newZipFileName, err)
					success = false
				}

				// Swap the names (read from new renamed file and write to the old name)
				outputZipFile = inputZipFile
				inputZipFile = newZipFileName

				// Original zip file exists so you can do field comparisons
				compareFields = true
			} else {
				// Use the default name for the output since nothing exists
				outputZipFile = newZipFileName
			}

		} else {
			// Create a new temp output file from the base name
			outputZipFile, err = buildFileName(parmBaseName, true)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error building output file '%s' using base name $%s: %s\n", outputZipFile, parmBaseName, err)
				return err
			}

			// Display the output file
			fmt.Fprintf(logWriter, "The output file is now %s\n", outputZipFile)

			// Set the input file
			inputZipFile = parmBaseName

			// Original zip file exists so you can do field comparisons
			compareFields = true
		}

	}

	// Check status
	if !success {
		fmt.Fprintf(os.Stderr, "Initializtion errors we encountered, processing terminated")
		return fmt.Errorf("initialization error")
	}

	// Success
	return nil
}

// process Do the mainline processing
func process() (err error) {
	// Load the file if not the first run
	if parmAnalyzeOnly {
		fmt.Fprintf(logWriter, "Performing analysis only, no zip file will be created\n")
	}

	if compareFields {
		fmt.Fprintf(logWriter, "Processing root dir against file %s\n", inputZipFile)
	}

	if parmExcludeCRC {
		fmt.Fprintf(logWriter, "No CRC will be computed or compared\n")
	}

	// If you are setup to compare fields, load the file
	if compareFields {
		var byteValues []byte

		// Uncompress from the last name
		byteValues, err = utils.UncompressFile(inputZipFile, outputFileName, zipPassword)

		if err != nil {
			fmt.Fprintf(logWriter, "Error uncompressing the previous zip file %s\n", inputZipFile)
			return err
		}

		if printStats {
			PrintMemoryStats("Uncompressed file")
		}

		err = loadFile(byteValues)

		if printStats {
			PrintMemoryStats("Loaded file")
		}

		if err != nil {
			fmt.Fprintf(logWriter, "Error loading the file '%s'\n", inputZipFile)
			return err
		}
	} else {
		if !parmVerifyExclude {
			fmt.Fprintf(logWriter, "Loading the first run from the root dir\n")
		}
	}

	// Walk the trees
	for _, RootDir := range rootDirs {
		// Walk the requested tree
		if walkFilepath {
			err = filepath.Walk(RootDir, walkTree)
		} else {
			err = godirwalk.Walk(RootDir, &godirwalk.Options{
				Callback: walkGoDir,
				ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
					// For the purposes of this example, a simple SkipNode will suffice,
					// although in reality perhaps additional logic might be called for.
					return godirwalk.SkipNode
				},
				Unsorted: true, // set true for faster yet non-deterministic enumeration (see godoc)
			})

		}

		if err != nil {
			fmt.Fprintf(logWriter, "Error initializing directory processing for %s\n", RootDir)
			return err
		}
	}

	// See if you need to save the file
	if parmVerifyExclude || parmAnalyzeOnly {
		// Just verifying or analyzing dont save
		err = nil
	} else {
		// Save the file
		err = saveFile()
	}

	// Success
	return err
}

// Walk the directory tree using filepath and process all files (not directories or links)
func walkTree(path string, info os.FileInfo, callerErr error) error {
	return processPath(path, info.IsDir(), info.Name(), filepath.SkipDir)
}

// Walk the directory tree using godirwalk and process all files (not directories or links)
func walkGoDir(path string, dirEntry *godirwalk.Dirent) error {
	return processPath(path, dirEntry.IsDir(), dirEntry.Name(), godirwalk.SkipThis)
}

// Common file processing regardless of directory parsing method
func processPath(path string, isDir bool, baseName string, skipDirReturn error) error {
	originalPath := path

	// Clean up the slashes
	path = strings.ReplaceAll(path, `\`, "/")

	// See if this name should be skipped
	skipEntry := false

	// Go through the exclude regular expressions for a match
	for _, regEx := range regexExcludes {
		if regEx.MatchString(strings.ToLower(baseName)) {
			// Flag the entry to skip and no need to keep going
			skipEntry = true
			break
		}
	}

	// Only process file names and not directories
	if isDir {
		// Check if you need to skip this entire directory
		if skipEntry {
			fmt.Fprintf(logWriter, "Skipping directory: %s\n", originalPath)
			return skipDirReturn
		}

		// Skip building a crc on just a directory name
		return nil
	}

	// Skip this file if requested
	if skipEntry {
		fmt.Fprintf(logWriter, "Skipping file: %s\n", originalPath)
		return nil
	}

	// Increment total file count
	totalFiles++

	// Skip processing if only evaluating excludes
	if parmVerifyExclude {
		return nil
	}

	// Compute the CRC64 for the specified file
	data, err := computeFileCRC64(path)

	if err != nil {
		fmt.Fprintf(logWriter, "Error processing file '%s': %s\n", originalPath, err)
		return err
	}

	// If not only building, then lookup the info and compare
	keyName := strings.ToLower(data.GetName())

	if compareFields {
		fileData, found := fileMap[keyName]

		// if it's found, then do the compare
		if found {
			// Check if the file is suspicious
			reason, suspicious := isSuspicious(data, fileData)

			if suspicious {
				fmt.Fprintf(logWriter, "Suspicious file %s: %s\n", originalPath, reason)
				suspiciousCt++
				data.SetSuspicious()
			} else {
				// Accumulate the number of changed records
				if !data.IsEqual(fileData) {
					data.SetMismatched()
					mismatchedEntries++
				} else {
					unchangedEntries++
				}
			}
		} else {
			// Not in the original file so count it as an add
			newEntries++
			data.SetAdded()
		}
	} else {
		// Increment the count
		newEntries++
		data.SetAdded()
	}

	// Update sizes
	totalFileSize = totalFileSize + data.GetSize()

	if data.GetSize() > maxFileSize {
		maxFileSize = data.GetSize()
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

		fileInfo := utils.FileInfo{}
		err := fileInfo.ParseCRCLine(currentLine)

		if err != nil {
			return err
		}

		// Clear the flag
		fileInfo.ClearFlag()

		// Add the entry to the map
		fileMap[strings.ToLower(fileInfo.GetName())] = fileInfo
	}

	// Return success
	return nil
}

// saveFile Save the intrnal map to a file by creating a file in an encrypted zip
func saveFile() error {
	// Create a buffer and write each entry from the map
	var buff bytes.Buffer

	var line string
	for _, entry := range fileMap {
		// Create the line for the file and write it into the buffer
		line = entry.BuildCRCLine()
		buff.WriteString(line)
	}

	// Build the compressed zip
	err := utils.CompressFile(outputZipFile, zipPassword, outputFileName, buff.Bytes())

	if err != nil {
		fmt.Fprintf(logWriter, "Error building the compressed zip %s: %s\n", outputZipFile, err)
		return err
	}

	// Return success
	return nil
}

// Send the email for the job finish, attached files as specified
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
			if !parmAnalyzeOnly {
				// Nothing to attach when analyzing
				attachFiles = append(attachFiles, outputZipFile)
			}
		default:
			fmt.Fprintf(logWriter, "Invalid attachment type for sending an email, %s\n", attachType)
		}
	}

	// Build the email subject
	emailSubject := fmt.Sprintf("%s scanned files: %d suspicious, %d total, %d added, %d modified %d deleted",
		parmHostname, suspiciousCt, totalFiles, newEntries, mismatchedEntries, deletedCt)
	// setup the message
	message := ""

	if parmAnalyzeOnly {
		message += "Analysis only specified, <b>no zip file produced</b><p>"
	}

	if err != nil {
		message += "Processing completed <b>in error</b><p>"
	} else {
		message += "Processing completed <b>without errors</b><p>"
	}

	// List attached files
	for _, files := range attachFiles {
		message += "File " + files + " attached<p>"
	}

	// Send the email
	err = utils.SendEmail(emailFrom, emailToList, emailCCList, emailSubject, message, attachFiles, emailCredentials)

	if err != nil {
		fmt.Fprintf(logWriter, "An error occurred sending the email: %s\n", err)
		return err
	}

	// Success
	return nil
}

// Print memory current usage statistics
func PrintMemoryStats(header string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Fprintf(logWriter, "%s: Alloc=%s MB, TotalAlloc=%s MB, Sys=%s MB, NumGC=%s\n",
		header,
		utils.NiceInt64(int64(memStats.Alloc/OneMB)),
		utils.NiceInt64(int64(memStats.TotalAlloc/OneMB)),
		utils.NiceInt64(int64(memStats.Sys/OneMB)),
		utils.NiceInt64(int64(memStats.NumGC/OneMB)))
}
