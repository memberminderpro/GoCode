package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"dacdb.com/GoCode/filecrc/utils"
	getopt "github.com/pborman/getopt/v2"
)

// JSON Configuration keyword constants
const (
	KwRootDirs    = "rootdirs"  // JSON Config root direcgtory
	KwDirSkips    = "dirskips"  // Pattern for skipping diretories and e erything below
	KwFileSkips   = "fileskips" // Pattern for skipping individual files
	KwZipName     = "zipname"   // JSON Config name of the zip file
	KwFileName    = "filename"  // JSON Config name of the file within the zip
	KwPassword    = "password"  // JSON Config zip password
	KwExclude     = "exclude"   // JSON Config List of files to exclude
	KwHostname    = "hostname"  // JSON Config host name for email subject
	KwSendMail    = "sendmail"  // JSON Config flag to indicate that email should be sent
	KwEmail       = "email"     // JSON Config email information
	KwEmailServer = "server"    // JSON Config email server name
	KwEmailPort   = "port"      // JSON Config email port number
	KwEmailUser   = "user"      // JSON Config email user name
	KwEmailCode   = "code"      // JSON Config email password
	KwEmailFrom   = "from"      // JSON Config email from
	KwEMailTo     = "to"        // JSON Config email to distribution
	KwEmailCC     = "cc"        // JSON Config cc distribution list
	KwEmailAttach = "attach"    // JSON Config switch to attach finished zip file
	KwLogFile     = "logfile"   // JSON Config log file name
	KwDebug       = "debug"     // Debug section
	KwDebugStats  = "stats"     // Print debug stats
	KwDebugMethod = "method"    // Tree walking method

	// Option flag specs
	FlagVerifyConfig  = 'v' // Flag to indicate config verification only
	FlagVerifyExclude = 'V' // Flag to indicate exclude file name only
	FlagConfigName    = 'c' // Flag to define the config file name
	FlagBaseName      = 'b' // Flag to indicate the base file name
	FlagAnalyzeOnly   = 'a' // Flag to inidicate analysis only (no output file)
	FlagExcludeCRC    = 'x' // Flag to indicate that CRC64 should NOT be performed
)

var (
	parmVerifyConfig  bool   = false // Verify only
	parmVerifyExclude bool   = false // Verify file selection criteria
	parmConfigName    string = ""    // Configuration file name
	parmBaseName      string = ""    // Base name to load for comparisons
	parmAnalyzeOnly   bool   = false // Analyze only flag
	parmExcludeCRC    bool   = false // Excluded CRC calculations
	parmHostname      string = ""    // Host name
)

// Other constants
const (
	DefaultOutputFileName = "fileinfo.txt" // Default name for the file inside the zip
	AttachLogName         = "log"          // Logical name for attaching the log file
	AttachZipName         = "zip"          // Logical name for attaching the zip file
)

func getParms(args []string) error {
	// Setup for processing
	flagSet := getopt.New()

	// Setup specs
	flagSet.Flag(&parmVerifyConfig, FlagVerifyConfig, "Verify configuration only")
	flagSet.Flag(&parmVerifyExclude, FlagVerifyExclude, "Verify config and exclude against directory")
	flagSet.Flag(&parmAnalyzeOnly, FlagAnalyzeOnly, "Analyze the data against the base, do not produce a new file")
	flagSet.Flag(&parmBaseName, FlagBaseName, "Optional name of the base file for comparisons")
	flagSet.Flag(&parmConfigName, FlagConfigName, "Name of the configuration file")
	flagSet.Flag(&parmExcludeCRC, FlagExcludeCRC, "Exclude CRC4 generation")

	// Parse the arguments
	err := flagSet.Getopt(args, nil)

	if err != nil {
		return err
	}

	// Do some verifications
	status := true

	if len(parmConfigName) == 0 {
		// See if possibly they used it as a positional parameter
		posArgs := flagSet.Args()
		if len(posArgs) == 1 {
			// Use the first positional parameter as the config name
			parmConfigName = posArgs[0]
		} else {
			fmt.Fprintf(os.Stderr, "The configuration file name is required\n")
			status = false
		}
	}

	// If verifying exclusions then also verify config
	if parmVerifyExclude {
		parmVerifyConfig = true
	}

	// Excluding crc may only be used with analyze
	if parmExcludeCRC && !parmAnalyzeOnly {
		fmt.Fprintf(os.Stderr, "You must also specify the analyze only flag when excluding crc calculations\n")
		status = false
	}

	if (parmVerifyExclude || parmVerifyConfig) && (parmAnalyzeOnly || parmExcludeCRC || len(parmBaseName) > 0) {
		fmt.Fprintf(os.Stderr, "Mutually exclusive parameters specified\n")
		fmt.Fprintf(os.Stderr, "-%c and -%c cannot be used with -%c -%c or -%c\n",
			FlagVerifyConfig, FlagVerifyExclude, FlagAnalyzeOnly, FlagExcludeCRC, FlagBaseName)
		status = false
	}

	// Check if you should move on
	if !status {
		return fmt.Errorf("processing terminated due to errors")
	}

	// Process the config file
	err = getConfig(parmConfigName)

	if err != nil {
		return err
	}

	// See if the base file specified exists
	if len(parmBaseName) > 0 {
		if _, err := os.Stat(parmBaseName); os.IsNotExist(err) {
			return fmt.Errorf("error processing the base name file '%s': %s", parmBaseName, err)
		}
	}

	// Return success if you get here
	return nil
}

// getParms Read and parse the JSON configuration file
func getConfig(configSpec string) error {
	// Read the config file
	config, err := ioutil.ReadFile(configSpec)

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
				rootDirs = append(rootDirs, strings.ToLower(dirName))
			}
		case KwZipName:
			inputZipFile = val.(string)
		case KwFileName:
			outputFileName = val.(string)
		case KwPassword:
			zipPassword = val.(string)
		case KwHostname:
			parmHostname = val.(string)
		case KwSendMail:
			emailFlag = val.(bool)
		case KwExclude:
			// Exclude dir names must start with a slash so it can be compared with the full path
			for _, spec := range val.([]interface{}) {
				excludes = append(excludes, spec.(string))
			}

		case KwEmail:
			// Initialzie credentials struct
			emailCredentials = utils.Credentials{}

			// Process the nested email JSON
			emailInfo := val.(map[string]interface{})
			for name, emailParm := range emailInfo {
				switch name {
				case KwEmailServer:
					emailCredentials.SetServer(emailParm.(string))
				case KwEmailPort:
					emailCredentials.SetPort(int(emailParm.(float64)))
				case KwEmailUser:
					emailCredentials.SetUserName(emailParm.(string))
				case KwEmailCode:
					emailCredentials.SetPassword(emailParm.(string))
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

		case KwDebug:
			debugInfo := val.(map[string]interface{})
			for name, debug := range debugInfo {
				switch name {
				case KwDebugStats:
					printStats = debug.(bool)

				case KwDebugMethod:
					var methodName = debug.(string)
					if strings.EqualFold(methodName, MethodFilepath) {
						walkFilepath = true
					} else if strings.EqualFold(methodName, MethodGoDir) {
						walkFilepath = false
					} else {
						fmt.Fprintf(os.Stderr, "Invalid directory walk method must be one of %s or %s", MethodGoDir, MethodFilepath)
						success = false
					}

				default:
					fmt.Fprintf(os.Stderr, "Invalid config file parameter %s\n", key)
					success = false
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

	if len(excludes) > 0 {
		// Compile each of the patterns
		for _, pattern := range excludes {
			regex, err := regexp.Compile(pattern)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Directory skip pattern '%s' is invalid\n", pattern)
				success = false
			} else {
				regexExcludes = append(regexExcludes, regex)
			}
		}
	}

	if len(inputZipFile) == 0 {
		fmt.Fprintf(os.Stderr, "A zip file name parameter, %s, is required\n", KwZipName)
		success = false
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
		if len(emailCredentials.GetServer()) == 0 {
			fmt.Fprintf(os.Stderr, "The email server, %s, was not specified\n", KwEmailServer)
			success = false
		}

		if emailCredentials.GetPort() == 0 {
			fmt.Fprintf(os.Stderr, "The email server port, %s, was not specified\n", KwEmailPort)
			success = false
		}

		if len(emailCredentials.GetUserName()) == 0 {
			fmt.Fprintf(os.Stderr, "The email user name, %s, was not specified\n", KwEmailUser)
			success = false
		}

		if len(emailCredentials.GetPassword()) == 0 {
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

		// Hostname is used as part of the email subject
		if len(parmHostname) == 0 {
			parmHostname, _ = os.Hostname()
		}
	}

	// Check email using example.com before continuing
	if emailFlag && success {
		var toList []string = []string{"test@example.com"}
		var ccList []string = make([]string, 0)
		var attachList []string = make([]string, 0)

		mailCheck := utils.SendEmail(emailFrom, toList, ccList, emailSubject, "A test message", attachList, emailCredentials)

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
