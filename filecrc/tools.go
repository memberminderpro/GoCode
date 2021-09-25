package main

import (
	"bytes"
	"fmt"
	"hash/crc64"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gophish/gomail"
	"github.com/malview/zip"
)

// Constants
const (
	DateTimeLayout = "2006-01-02 15:04:05" // Formatting for yyyy-mm-dd HH:mm:ss
	FieldSep       = "|"                   // Output file field separator
)

// EmailCredentials Email credentials
type credentials struct {
	userName string // Email server login name
	password string // EMail server password
	server   string // Email server name
	port     int    // EMail port number
}

// CRCInfo Info for a file CRC
type crcInfo struct {
	name     string    // Name of the file
	modified time.Time // Last accessed date as a string MM/dd/yyyy hh:mm:ss
	size     int64     // File size
	crc      uint64    // Computed CRC
}

// compressFile Create an encrypted zip with a file from string contents
// zipName:	Name of the zip file to create
// password: Encryption password (empty if no emcryption)
// fileName: Name of the encrypted file in the zip file
// contents: A byte buffer of the contents to write
func compressFile(zipName, password, fileName string, contents []byte) error {
	//Create the zip file
	fileWriter, err := os.Create(zipName)

	if err != nil {
		return err
	}

	defer fileWriter.Close()

	// Create the zip writer for the file
	var zipWriter io.Writer
	zipper := zip.NewWriter(fileWriter)

	// Create an encrypted ZIP if a password was provided
	if len(password) == 0 {
		zipWriter, err = zipper.Create(fileName)
	} else {
		zipWriter, err = zipper.Encrypt(fileName, password, zip.AES256Encryption)
	}

	if err != nil {
		return err
	}

	defer zipper.Close()

	// Create an output buffer and write the content there
	_, err = io.Copy(zipWriter, bytes.NewReader(contents))

	if err != nil {
		return err
	}

	// Good return
	return nil
}

// uncompressFile Extract a file from an encrypted zip
// zipName: Name of the zip file
// fileName: Name of the file in the zip to extract
// password: Encryption password
// returns a byte slice containing the unencrypted uncompressed data
func uncompressFile(zipName, fileName, password string) ([]byte, error) {
	// read the password zip
	rdr, err := os.Open(zipName)

	if err != nil {
		return nil, err
	}

	// Get the file size and setup a reader
	info, _ := rdr.Stat()
	size := info.Size()
	zipReader, err := zip.NewReader(rdr, size)

	if err != nil {
		return nil, err
	}

	// FInd the requested file and read into a string
	for _, entry := range zipReader.File {
		if entry.Name != fileName {
			continue
		}

		// Set the password and read into a buffer
		if len(password) > 0 {
			entry.SetPassword(password)
		}

		fileReader, err := entry.Open()

		if err != nil {
			return nil, err
		}

		defer fileReader.Close()

		var buf bytes.Buffer
		_, err = io.Copy(&buf, fileReader)

		if err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}

	// Can't find the requested file in the zip
	return nil, fmt.Errorf("the file '%s' is not in the zip file '%s'", fileName, zipName)
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

// computeFileCRC64 Compute the CRC64 for the file specified in the path
// path: Full pathname of the file to compute the crc64 for
// returns CRC64 for the file contents
func computeFileCRC64(path string) (crcInfo, error) {
	// Setup an empty response for errors
	response := crcInfo{}

	// Open the file

	fileRdr, err := os.Open(path)

	if err != nil {
		return response, err
	}

	//defer fileRdr.Close()

	// Read the file in and compute the crc
	content, err := ioutil.ReadAll(fileRdr)

	if err != nil {
		return response, err
	}

	// Prepare the response
	info, err := fileRdr.Stat()

	if err != nil {
		return response, err
	}

	// Compute the crc
	response.crc = crc64.Checksum(content, crc64.MakeTable(crc64.ECMA))

	// Set the name and size
	response.name = path
	response.size = info.Size()

	// Format the date
	response.modified = info.ModTime()

	// Cleanup
	err = fileRdr.Close()

	if err != nil {
		return response, err
	}

	content = nil

	// Return result
	return response, nil
}

// buildCRCLine  Construct a CRC summary line with a separator
func buildCRCLine(info crcInfo) string {
	return fmt.Sprintf("%s%s%s%s%d%s%d\n", info.name, FieldSep, info.modified.Format(DateTimeLayout), FieldSep, info.size, FieldSep, info.crc)
}

// parseCRCLine Parse a line of file status into a CRCInfo
func parseCRCLine(line string) (crcInfo, error) {
	// Remove trailing newline if there
	line = strings.TrimSuffix(line, "\n")

	// Split the string first
	parts := strings.Split(line, FieldSep)

	if len(parts) != 4 {
		return crcInfo{}, fmt.Errorf("the line '%s' is invalid", line)
	}

	// Build the info
	var err error = nil
	var dummy = crcInfo{}

	info := crcInfo{}

	info.name = parts[0]
	info.modified, err = time.Parse(DateTimeLayout, parts[1])

	if err != nil {
		return dummy, err
	}
	info.size, err = strconv.ParseInt(parts[2], 10, 64)

	if err != nil {
		return dummy, err
	}

	info.crc, err = strconv.ParseUint(parts[3], 10, 64)

	if err != nil {
		return dummy, err
	}

	// Success
	return info, nil
}

// buildFileName Builds a filename based on the suggested name.  If it exists, a numeric suffix is added until unique
func buildFileName(fileName string) (string, error) {
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
	if !found {
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
