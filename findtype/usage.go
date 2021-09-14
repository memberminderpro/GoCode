package main

import (
	"fmt"
	"os"
)

// usage Display program usage
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: [-v] filecrc config.json {previousFileName}\n")
	fmt.Fprintf(os.Stderr, "  use -v to verify the configuration file contents\n\n")
	fmt.Fprintf(os.Stderr, "Sample JSON:\n%s\n",
		`
{
    "rootdirs"   : ["c:/Dir1", "c:/Dir2"],
    "extensions" : ["ext1", "ext2"],
    "sendmail"   : true,
    "logfile"    : "findtype.log",
	"logging"    : true,
	"hostname"   : "MyHost",
    "email"      : 
	{
        "server"  : "mail.server.net",
        "port"    : 587,
        "user"    : "dacdb@example.com",
        "code"    : "secret",
        "from"    : "talend@dacdb.com",
        "to"      : ["peter@dacdb.com"],
        "cc"      : [],    
	}
}`)

	fmt.Fprintf(os.Stderr, "Keywords\n")
	fmt.Fprintf(os.Stderr, "%s: Specifies a list of root diretories to scan \n", KwRootDirs)
	fmt.Fprintf(os.Stderr, "%s: Specifies the list of file name extsnsions to check for\n", KwExtensions)
	fmt.Fprintf(os.Stderr, "%s: Specifies an optional log output file\n", KwLogFile)
	fmt.Fprintf(os.Stderr, "%s: Specifies if file names should be logged (default: true)\n", KwLogNames)
	fmt.Fprintf(os.Stderr, "%s: Specifies the name of the host\n", KwHostName)
	fmt.Fprintf(os.Stderr, "%s: Specifies whether to send an email or not\n", KwSendMail)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server name\n", KwEmailServer)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server port\n", KwEmailPort)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server login\n", KwEmailUser)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email server password\n", KwEmailCode)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email sender email address\n", KwEmailFrom)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email distribution list\n", KwEMailTo)
	fmt.Fprintf(os.Stderr, "%s: Specifies the email CC distribution list\n", KwEmailCC)
	fmt.Fprintf(os.Stderr, "%s: Specifies an optional log output file (default: stdout)\n", KwLogFile)
}
