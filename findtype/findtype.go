package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gophish/gomail"
)

var (
	extensions       map[string]int = make(map[string]int) // Count of extensions encountered
	rootDirs         []string       = make([]string, 0)    // Starting directories for scan
	fileCt           int            = 0                    // numner of files scanned
	dirCt            int            = 0                    // Number of directories scanned
	extCt            int            = 0                    // Total ext matches
	skipCt           int            = 0                    // Number of skipped files
	emailFlag        bool           = false                // Flag to send emails
	logNamesFlag     bool           = true                 // FLag to indicate if there should be file name logging
	emailCredentials credentials                           // Credentials for accessing email
	emailFrom        string                                // Email from email address
	emailToList      []string       = make([]string, 0)    // To distribution list
	emailCCList      []string       = make([]string, 0)    // Email CC distribution list
	emailSubject     string                                // Email subject
	logFileName      string                                // Name for logging (defaults to stderr)
	logWriter        *os.File       = os.Stdout            // Default log output
)

// EmailCredentials Email credentials
type credentials struct {
	userName string // Email server login name
	password string // EMail server password
	server   string // Email server name
	port     int    // EMail port number
}

func main() {
	// Initialize and check status
	parseErrs := getParms()

	if parseErrs != nil {
		fmt.Fprintln(os.Stderr, parseErrs)
		usage()
		os.Exit(2)
	}

	// Close the log file at termination if not stdout
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
	printStats(logWriter, "\n")

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
		return fmt.Errorf("initialization error")
	}

	// Success
	return nil
}

func process() error {
	// Walk the trees
	for _, rootDir := range rootDirs {
		fmt.Fprintf(logWriter, "Processing directory %s\n", rootDir)

		// Recursively walk the trees
		err := filepath.Walk(rootDir, walkTree)

		if err != nil {
			return fmt.Errorf("error initializing directory processing for %s", rootDir)
		}
	}

	return nil
}

func processEmail(err error) error {
	// Make sure we're supposed send the email
	if !emailFlag {
		return nil
	}

	// Build list of file attachments
	attachFiles := []string{logFileName}

	// setup the message
	var buff bytes.Buffer

	// Write directories processed
	fmt.Fprintf(&buff, "Directories processed: %v<p>", rootDirs)

	// Print stats using HTML
	printStats(&buff, "<p>")

	message := buff.String()

	if err != nil {
		message += "Processing completed <b>in error</b><p>"
	} else {
		message += "Processing completed <b>without errors</b><p>"
	}

	// Build the extension name list
	var extList string
	for key := range extensions {
		if len(extList) > 0 {
			extList += ", "
		}

		// Add the name to the list
		extList += key
	}

	// Build the email subject
	emailSubject := fmt.Sprintf("Found %d file extensions from %s in %v", extCt, extList, rootDirs)

	// Send the email
	err = sendEmail(emailFrom, emailToList, emailCCList, emailSubject, message, attachFiles, emailCredentials)

	if err != nil {
		fmt.Fprintf(logWriter, "An error occurred sending the email: %s\n", err)
		return err
	}

	// Success
	return nil
}

func printStats(log io.Writer, newLine string) {
	// Print summary stats
	fmt.Fprintf(log, "Directories scanned: %d%s", dirCt, newLine)
	fmt.Fprintf(log, "Files checked: %d%s", fileCt, newLine)
	fmt.Fprintf(log, "Files skipped: %d%s", skipCt, newLine)

	fmt.Fprintf(log, "Extensions found: %d%s", extCt, newLine)

	// Print the extensions found
	for ext, ct := range extensions {
		fmt.Fprintf(log, "  Extension %s: %d%s", ext, ct, newLine)
	}
}
func walkTree(path string, info os.FileInfo, callerErr error) error {
	// If you can't access the info, skip the file
	if info == nil || callerErr != nil {
		skipCt++
		return nil
	}

	// Only process file names and not directories
	if info.IsDir() {
		// Skip proecssing just a directory name
		dirCt++
		return nil
	}

	// Increment file ct
	fileCt++

	// Find the last dot for the suffix
	lastDot := strings.LastIndex(path, ".")

	// Get the suffix if it exists
	if lastDot+1 < len(path) {
		suffix := strings.ToLower(path[lastDot+1:])
		if _, found := extensions[suffix]; found {
			// Log it, if requested
			if logNamesFlag {
				fmt.Fprintf(logWriter, "%s\n", path)
			}

			// Increment counts
			extensions[suffix]++
			extCt++
		}
	}

	// Done with this file, return
	return nil
}

// sendEmail Send an HTML email to the designated recipients
// from: The email address of the sender
// to: A slice of email addresses to send the content to
// cc: An optional slice of email addresses to send to
// subject: A string indicating the message
// message: The HTML message content
// attachments: A slice of file names for files to be attached
// loginInfo: Login info for the mail server as a credentials structure
func sendEmail(from string, to []string, cc []string, subject, message string, attachments []string, loginInfo credentials) error {
	// Create a mail message
	m := gomail.NewMessage()

	// Set the header fields
	m.SetHeader("From", from)
	m.SetHeader("To", to...)

	if len(cc) > 0 {
		m.SetHeader("Cc", cc...)
	}

	m.SetHeader("Subject", subject)
	m.SetBody("text/html", message)

	// Attach the specified files
	for _, fileName := range attachments {
		m.Attach(fileName)
	}

	// Send the email
	d := gomail.NewDialer(loginInfo.server, loginInfo.port, loginInfo.userName, loginInfo.password)

	// Send the email to Bob, Cora and Dan.
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	// Return success
	return nil
}
