package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/gophish/gomail"
	"github.com/malview/zip"
)

// EmailCredentials Email credentials
type Credentials struct {
	userName string // Email server login name
	password string // EMail server password
	server   string // Email server name
	port     int    // EMail port number
}

// Setters
func (c *Credentials) SetUserName(value string) {
	c.userName = value
}

func (c *Credentials) SetPassword(value string) {
	c.password = value
}

func (c *Credentials) SetServer(value string) {
	c.server = value
}

func (c *Credentials) SetPort(value int) {
	c.port = value
}

// Getters
func (c *Credentials) GetUserName() string {
	return c.userName
}

func (c *Credentials) GetPassword() string {
	return c.password
}

func (c *Credentials) GetServer() string {
	return c.server
}

func (c *Credentials) GetPort() int {
	return c.port
}

// compressFile Create an encrypted zip with a file from string contents
// zipName:	Name of the zip file to create
// password: Encryption password (empty if no emcryption)
// fileName: Name of the encrypted file in the zip file
// contents: A byte buffer of the contents to write
func CompressFile(zipName, password, fileName string, contents []byte) error {
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
// fileName: Name of the file in the zip to extract (or empty string for the first file)
// password: Encryption password
// returns a byte slice containing the unencrypted uncompressed data
func UncompressFile(zipName, fileName, password string) ([]byte, error) {
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

	// Find the requested file and read into a string
	for _, entry := range zipReader.File {
		// Check if the name is the one specified
		if len(fileName) > 0 && entry.Name != fileName {
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
func SendEmail(from string, to []string, cc []string, subject, message string, attachments []string, loginInfo Credentials) error {
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

func NiceInt(value int) string {
	return NiceInt64(int64(value))
}

func NiceInt64(value int64) string {
	return humanize.Comma(value)
}
