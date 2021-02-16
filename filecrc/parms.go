package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// JSON Configuration keyword constants
const (
	KwRootDir      = "rootdir"  // JZON Config root direcgtory
	KwZipName      = "zipname"  // JSON Config name of the zip file
	KwFileName     = "filename" // JSON Config name of the file within the zip
	KwPassword     = "password" // JSON Config zip password
	KwExclude      = "exclude"  // JSON Config List of files to exclude
	KwSendMail     = "sendmail" // JSON Config flag to indicate that email should be sent
	KwEmail        = "email"    // JSON Config email information
	KwEmailServer  = "server"   // JSON Config email server name
	KwEmailPort    = "port"     // JSON Config email port number
	KwEmailUser    = "user"     // JSON Config email user name
	KwEmailCode    = "code"     // JSON Config email password
	KwEmailFrom    = "from"     // JSON Config email from
	KwEMailTo      = "to"       // JSON Config email to distribution
	KwEmailCC      = "cc"       // JSON Config cc distribution list
	KwEmailSubject = "subject"  // JSON Config email subject
	KwEmailAttach  = "attach"   // JSON Config switch to attach finished zip file
	KwLogFile      = "logfile"  // JSON Config log file name
)

// Other constants
const (
	DefaultOutputFileName = "crcinfo.txt" // Default name for the file inside the zip
	AttachLogName         = "log"         // Logical name for attaching the log file
	AttachZipName         = "zip"         // Logical name for attaching the zip file
)

// getParms Read and parse the JSON configuration file
func getParms() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("You must specify a configuration file name")
	}

	// Read the config file
	config, err := ioutil.ReadFile(os.Args[1])

	if err != nil {
		return err
	}

	// Unmarshal the JSON source into a map[string]interface{}
	argMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(config), &argMap)

	// Make sure it worked
	if err != nil {
		return err
	}

	// Initialize
	success := true

	// Process JSON
	for key, val := range argMap {
		switch strings.ToLower(key) {
		case KwRootDir:
			// The root dir should only uae forward clashes and NOT end in one for comparisons
			rootDir = strings.ReplaceAll(val.(string), `\`, "/")
			if strings.HasSuffix(rootDir, "/") {
				// Remove the trailing slash
				rootDir = rootDir[1:]
			}
			// Set the size
			rootDirSize = len(rootDir)
		case KwZipName:
			zipFileName = val.(string)
		case KwFileName:
			outputFileName = val.(string)
		case KwPassword:
			zipPassword = val.(string)
		case KwSendMail:
			emailFlag = val.(bool)
		case KwExclude:
			// Exclude dir names must start with a slash so it can be compared with the full path
			for _, filename := range val.([]interface{}) {
				dirName := strings.ReplaceAll(filename.(string), `\`, "/")

				// Add a slash if not there
				if !strings.HasPrefix(dirName, "/") {
					dirName = "/" + dirName
				}

				// Add to the map of xcludes
				ex := strings.ToLower(dirName)
				excludes[ex] = nil
			}
		case KwEmail:
			// Initialzie credentials struct
			emailCredentials = credentials{}

			// Process the nested email JSON
			emailInfo := val.(map[string]interface{})
			for name, emailParm := range emailInfo {
				switch name {
				case KwEmailServer:
					emailCredentials.server = emailParm.(string)
				case KwEmailPort:
					emailCredentials.port = int(emailParm.(float64))
				case KwEmailUser:
					emailCredentials.userName = emailParm.(string)
				case KwEmailCode:
					emailCredentials.password = emailParm.(string)
				case KwEmailFrom:
					emailFrom = emailParm.(string)
				case KwEMailTo:
					for _, toName := range emailParm.([]interface{}) {
						emailToList = append(emailToList, toName.(string))
					}
				case KwEmailCC:
					for _, ccName := range emailParm.([]interface{}) {
						emailCCList = append(emailCCList, ccName.(string))
					}
				case KwEmailSubject:
					emailSubject = emailParm.(string)
				case KwEmailAttach:
					for _, attachTypes := range emailParm.([]interface{}) {
						kind := strings.ToLower(attachTypes.(string))
						switch kind {
						case AttachLogName:
							fallthrough
						case AttachZipName:
							emailAttachments = append(emailAttachments, kind)
						default:
							fmt.Fprintf(os.Stderr, "The attachment keyword '%s' is invalid\n", kind)
						}
					}
				default:
					success = false
					fmt.Fprintf(os.Stderr, "The email keyword '%s' is invalid\n", name)
				}
			}
		case KwLogFile:
			logFileName = val.(string)
		default:
			fmt.Fprintf(os.Stderr, "Invalid config file parameter %s\n", key)
			success = false
		}
	}

	// Verify parms
	if len(rootDir) == 0 {
		fmt.Fprintf(os.Stderr, "The root directory name parameter, %s, is required\n", KwRootDir)
		success = false
	}

	if len(zipFileName) == 0 {
		fmt.Fprintf(os.Stderr, "A zip file name parameter, %s, is required\n", KwZipName)
		success = false
	}

	if len(outputFileName) == 0 {
		fmt.Fprintf(logWriter, "The output file name was not spcified, %s us being used\n", DefaultOutputFileName)
	}

	if len(zipPassword) == 0 {
		fmt.Fprintf(logWriter, "No zip encrption password supplied, the file will not be encrypted\n")
	}

	if emailFlag {
		// Validate email configuration
		if len(emailFrom) == 0 {
			fmt.Fprintf(os.Stderr, "The email from address, %s, was not specified\n", KwEmailFrom)
			success = false
		}

		if len(emailToList) == 0 {
			fmt.Fprintf(os.Stderr, "The email to list, %s, was not specified\n", KwEMailTo)
			success = false
		}

		if len(emailSubject) == 0 {
			fmt.Fprintf(os.Stderr, "The email subject, %s, was not specified\n", KwEmailSubject)
			success = false
		}

		// Check email credentials
		if len(emailCredentials.server) == 0 {
			fmt.Fprintf(os.Stderr, "The email server, %s, was not specified\n", KwEmailServer)
			success = false
		}

		if emailCredentials.port == 0 {
			fmt.Fprintf(os.Stderr, "The email server port, %s, was not specified\n", KwEmailPort)
			success = false
		}

		if len(emailCredentials.userName) == 0 {
			fmt.Fprintf(os.Stderr, "The email user name, %s, was not specified\n", KwEmailUser)
			success = false
		}

		if len(emailCredentials.password) == 0 {
			fmt.Fprintf(os.Stderr, "The email login password, %s, was not specified\n", KwEmailCode)
			success = false
		}

		// If attachments specify the log, theere has to be a log spec too
		for _, attachType := range emailAttachments {
			if attachType == AttachLogName {
				if len(logFileName) == 0 {
					fmt.Fprintf(os.Stderr, "Emailattachment of the log file was specified but no log file was defined\n")
					success = false
				}
			}
		}
	}

	// Check email using example.com before continuing
	if emailFlag && success {
		var toList []string = []string{"test@example.com"}
		var ccList []string = make([]string, 0)
		var attachList []string = make([]string, 0)

		mailCheck := sendEmail(emailFrom, toList, ccList, emailSubject, "A test message", attachList, emailCredentials)

		if mailCheck != nil {
			success = false
			fmt.Fprintf(os.Stderr, "Error validating email connection: %s\n", mailCheck)
		}
	}

	// Return
	if !success {
		return fmt.Errorf("Errors was found in processing the configuration")
	}

	// success
	return nil
}