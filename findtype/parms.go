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
	KwRootDirs    = "rootdirs"   // JSON Config root direcgtory
	KwExtensions  = "extensions" // JSON Config list of extensions to check
	KwSendMail    = "sendmail"   // JSON Config flag to indicate that email should be sent
	KwLogFile     = "logfile"    // JSON Config log file name
	KwEmail       = "email"      // JSON Config email information
	KwEmailServer = "server"     // JSON Config email server name
	KwEmailPort   = "port"       // JSON Config email port number
	KwEmailUser   = "user"       // JSON Config email user name
	KwEmailCode   = "code"       // JSON Config email password
	KwEmailFrom   = "from"       // JSON Config email from
	KwEMailTo     = "to"         // JSON Config email to distribution
	KwEmailCC     = "cc"         // JSON Config cc distribution list
	KwEmailAttach = "attach"     // JSON Config switch to attach finished zip file
)

// Other constants
var DefaultOutputFileName = "logfile.txt" // Default name for the file inside the zip

// getParms Read and parse the JSON configuration file
func getParms() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("you must specify a configuration file name")
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
		case KwRootDirs:
			for _, filename := range val.([]interface{}) {
				// CHange backslash to slash and remove any trailing slash
				dirName := strings.TrimSuffix(strings.ReplaceAll(filename.(string), `\`, "/"), "/")

				// Add to the list of root directories
				rootDirs = append(rootDirs, dirName)
			}

		case KwSendMail:
			emailFlag = val.(bool)

		case KwExtensions:
			// Exclude dir names must start with a slash so it can be compared with the full path
			for _, value := range val.([]interface{}) {
				// Make it a string
				extName := value.(string)

				// Remove a leading period is there
				extName = strings.TrimPrefix(extName, ".")

				// Add to the map of extensions and initialize to zero
				key := strings.ToLower(extName)
				extensions[key] = 0
			}

		case KwLogFile:
			logFileName = val.(string)

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
				default:
					success = false
					fmt.Fprintf(os.Stderr, "The email keyword '%s' is invalid\n", name)
				}
			}
		default:
			fmt.Fprintf(os.Stderr, "Invalid config file parameter %s\n", key)
			success = false
		}
	}

	// Verify parms
	if len(rootDirs) == 0 {
		fmt.Fprintf(os.Stderr, "The root directory name parameter, %s, is required\n", KwRootDirs)
		success = false
	}

	if len(extensions) == 0 {
		fmt.Fprintf(os.Stderr, "The list of extensions parameter, %s, is required\n", KwExtensions)
		success = false
	}

	if len(logFileName) == 0 {
		fmt.Fprintf(logWriter, "The output file name was not spcified, %s us being used\n", DefaultOutputFileName)

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
		return fmt.Errorf("errors was found in processing the configuration")
	}

	// success
	return nil
}
