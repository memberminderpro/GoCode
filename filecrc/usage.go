package main

import (
	"fmt"
	"os"
)

// usage Display program usage
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: filecrc [-v] config.json {previousFileName}\n")
	fmt.Fprintf(os.Stderr, "  use -v to verify the configuration file contents\n\n")
	fmt.Fprintf(os.Stderr, "Sample JSON:\n%s\n",
		`
{
    "rootdirs"  : ["c:/Dir1", "c:/Dir2"],
    "zipname"   : "filesinfo.zip",
    "filename"  : "stats.txt",
    "password"  : "dacdbrocks",
    "exclude"   : ["file1", "file2"],
    "sendmail"  : true,
    "logfile" : "filecrc.log",
    "email"     : {
        "server"  : "mail.server.net",
        "port"    : 587,
        "user"    : "dacdb@example.com",
        "code"    : "secret",
        "from"    : "talend@dacdb.com",
        "to"      : ["peter@dacdb.com"],
        "cc"      : [],
        "subject" : "Filecrc generation status",
        "attach"  : ["log", "zip"]
    },
	"debug"     : {
		"stats"   : true,
		"method"  : "godirwalk"
	}
}`)

	fmt.Fprintf(os.Stderr, "Keywords\n")
	fmt.Fprintf(os.Stderr, "%s: Specifies a list of directories to scan\n", KwRootDirs)
	fmt.Fprintf(os.Stderr, "%s: Specifies the name of the zip file\n", KwZipName)
	fmt.Fprintf(os.Stderr, "%s: Specifies the name of the file inside the zip file\n", KwFileName)
	fmt.Fprintf(os.Stderr, "%s: Specifies the zip encryption password\n", KwPassword)
	fmt.Fprintf(os.Stderr, "%s: Specifies an optional list of directories to exclude\n", KwExclude)
	fmt.Fprintf(os.Stderr, "%s: Specifies whether to send an email or not\n", KwSendMail)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server name\n", KwEmailServer)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server port\n", KwEmailPort)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server login\n", KwEmailUser)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server password\n", KwEmailCode)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email sender email address\n", KwEmailFrom)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email distribution list\n", KwEMailTo)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email CC distribution list\n", KwEmailCC)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email subject\n", KwEmailSubject)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email a list of logical files to attach\n", KwEmailAttach)
	fmt.Fprintf(os.Stderr, "    note: valid names are \"%s\", \"%s\"\n", AttachLogName, AttachZipName)
	fmt.Fprintf(os.Stderr, "%s: Specifies an optional log output file (default: stdout)\n", KwLogFile)
}
