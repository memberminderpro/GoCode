package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Configuration JSON constants
const (
	KwRoot     = "webroot"
	KwPaths    = "paths"
	KwVars     = "vars"
	KwExcludes = "exclude"
	KwSkipDirs = "skipdirs"
	KwSave     = "save"
	KwVerbose  = "verbose"
)

// Regular expression to isolate cffunction (with name=) and cfinvoke (with component= and method=)
var regExp = regexp.MustCompile(`<([Cc][Ff][Ff][Uu][Nn][Cc][Tt][Ii][Oo][Nn]|[Cc][Ff][Ii][Nn][Vv][Oo][Kk][Ee])\s+` +
	`|` +
	`(\s*([Cc][Oo][Mm][Pp][Oo][Nn][Ee][Nn][Tt]|[Mm][Ee][Tt][Hh][Oo][Dd]|[Nn][Aa][Mm][Ee])\s*=\s*"([^"]*)")`)

// Regular expression for matching the base part of the fle name
var regexpFName = regexp.MustCompile(`^[a-zA-Z].*\.([Cc][Ff][Cc]|[Cc][Ff][Mm])$`)

// Regular expression to extract the variable names in between hashes
var findVars = regexp.MustCompile(`(#[^#]*#)`)

// Component definitions
type compDef struct {
	name  string             // Full name of the component (without lower case)
	funcs map[string]funcDef // Function definitions
}

// Function definition
type funcDef struct {
	name   string               // Name for this function
	usedBy map[string]funcUsage // Where this function is called from
}

// Usage data
type funcUsage struct {
	cleanName string // Just a trimmed version of the file name
	useLines  []int  // Line numbers in this file
}

// Deferred cfinvoke information
type defInvoke struct {
	fileName  string // Name of the file this info was taken from
	line      int    // Line number the cfinvoke occurred on
	component string // Name of the component being invoked
	method    string // Name of the method in the component
}

// Collection of invoke information for processing after all cffunctions have been found
var deferredList = make([]defInvoke, 0, 10000)

// Container for cross references
var xref = make(map[string]compDef, 1000)

// Collect orphans by component and method
var orphans map[string][]string = make(map[string][]string) // Collect orphan functions by component

// Output directions
var orphanWriter *os.File = os.Stderr  // Default orphan output
var missingWriter *os.File = os.Stderr // Default Misssing output
var logWriter *os.File = os.Stdout     // Log output file

// Buffer for file scanning
var scanBuff = make([]byte, 5000000)

// Number of functions captured
var totalFuncs = 0

// Runtime parameters
var rootDir string                                                 // The root directory to start scanning (known as CF webroot)
var rootDirSize int                                                // Size of the root dir specification
var tagPath []string = make([]string, 0)                           // List of directories for custom tag directories to search for components
var excludes []string = make([]string, 0)                          // List of fully wualified files to exclude
var variables map[string]string = make(map[string]string)          // Map of variable names and replacement text
var skipDirs map[string]interface{} = make(map[string]interface{}) // Directories to skip
var verboseMode bool = false                                       // Verbose output setting
var crossRefNames []string = make([]string, 0)                     // Array of names to produce a cross reference for

// Counters
var missingFuncCt int = 0   // Number of missing functions
var missingMethodCt int = 0 // Number of missing methods
var orphanCompCt int = 0    // Number of orphan components
var orphanFuncCt int = 0    // Number of orphan functions

// For debugging - current line from the scanner
var currentLine string

func main() {
	timeStart := time.Now().Unix()
	fmt.Fprintln(os.Stdout, "Beginning build process")

	parseErrs := getParms()

	if parseErrs != nil {
		fmt.Fprintln(os.Stderr, parseErrs)
		pgmUsage()
		os.Exit(2)
	}

	// Close files that are not system output when done
	if missingWriter != os.Stderr {
		defer missingWriter.Close()
	}

	if orphanWriter != os.Stderr {
		defer orphanWriter.Close()
	}

	if logWriter != os.Stdout {
		defer logWriter.Close()
	}

	// Process the file structure recursively
	err := filepath.Walk(rootDir, walkTree)

	if err != nil {
		fmt.Fprintln(logWriter, err)
		os.Exit(2)
	}

	timeBuild := time.Now().Unix()
	fmt.Fprintln(os.Stdout, "Beginning analysis and reporting")

	// Process list of deferred cfinvoke calls
	processInvoke()

	// Find list of orphan components/methods
	processOrphans()
	displayOrphans()

	// Display cross references
	for _, componentName := range crossRefNames {
		if strings.EqualFold(componentName, "all") {
			fmt.Fprintf(logWriter, "The 'all' cross reference option is not yet supported")
			fmt.Fprintf(os.Stderr, "The 'all' cross reference option is not yet supported")
			break
		}

		// Generate cross reference
		crossReference(componentName)
	}

	fmt.Fprintf(logWriter, "\n")
	fmt.Fprintf(logWriter, "There are %d components defined with a total of %d functions\n", len(xref), totalFuncs)
	fmt.Fprintf(logWriter, "There are %d cfinvoke calls processed\n", len(deferredList))
	fmt.Fprintf(logWriter, "Number of missing referenced functions: %d\n", missingFuncCt)
	fmt.Fprintf(logWriter, "Number of missing referenced methods: %d\n", missingMethodCt)
	fmt.Fprintf(logWriter, "Number of orphaned components: %d\n", orphanCompCt)
	fmt.Fprintf(logWriter, "Number of orphaned functions: %d\n", orphanFuncCt)
	fmt.Fprintln(logWriter, "Processing completed successfully")

	timeFinish := time.Now().Unix()
	fmt.Fprintf(logWriter, "Build time: %d seconds, Analysis and reporting: %d seconds, total: %d\n", timeBuild-timeStart, timeFinish-timeBuild, timeFinish-timeStart)
	fmt.Fprintf(os.Stdout, "Build time: %d seconds, Analysis and reporting: %d seconds, total: %d\n", timeBuild-timeStart, timeFinish-timeBuild, timeFinish-timeStart)
	fmt.Fprintln(os.Stdout, "Processing completed successfully")
}

// pgmUsage Display sample usage
func pgmUsage() {
	fmt.Fprintf(os.Stderr, "Usage: cfxref config.json {list of components to display cross reference}\n\n")
	fmt.Fprintf(os.Stderr, "Sample JSON:\n%s\n",
		`
{
    "verbose"   : true,
    "webroot"   : "c:/Development/Rotary_CURRENT",
    "paths"     : ["/CFC", "/CFC2"],
    "vars"      : {"APPLICATION.DIR":"/CFC", "APPLICATION.SITECFCDIRECTORY": "/CFC"},
    "exclude"   : ["/Application.cfc"],
    "skipdirs"  : ["Dir/OldFiles", "Dir2/OldFiles"],
    "save"      : {"missing":"missing.txt", "orphans":"orphans.txt", "log":"log.txt"}
}`)

	fmt.Fprintf(os.Stderr, "Keywords\n")
	fmt.Fprintf(os.Stderr, "%s: a flag to enable verbose logging (boolean: true|false)\n", KwRoot)
	fmt.Fprintf(os.Stderr, "%s: The fully qualified web root directory\n", KwRoot)
	fmt.Fprintf(os.Stderr, "%s: An array of path names relative to the web root\n", KwPaths)
	fmt.Fprintf(os.Stderr, "%s: A set of json varname=value specifications\n", KwVars)
	fmt.Fprintf(os.Stderr, "%s: An array of cfc names relative to the root (i.e. /Application.cfc\n", KwExcludes)
	fmt.Fprintf(os.Stderr, "%s: An array of directory names relative to the root (i.e. /Application.cfc\n", KwSkipDirs)
	fmt.Fprintf(os.Stderr, "%s: A set of JSON variables for outputtingdata\nVariables are 'missing' and 'orphans' (default is display)\n", KwSave)
	fmt.Fprintf(os.Stderr, "NOTE: By Default the directory .svn is always skipped\n")
}

// getParms Get the parms from the command line argument and process
func getParms() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("A JSON configuration file must be specified")
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

	// Check for errors
	passed := true

	// Since the config may use wrong case values, iterate doing case insensitive processing
	var missingFileName string
	var orphanFileName string
	var logFileName string

	for key, val := range argMap {
		switch strings.ToLower(key) {
		case KwVerbose:
			verboseMode = val.(bool)
		case KwRoot:
			rootDir = val.(string)
		case KwPaths:
			values := val.([]interface{})
			for _, path := range values {
				tagPath = append(tagPath, cleanDirName(path.(string)))
			}
		case KwExcludes:
			values := val.([]interface{})
			for _, name := range values {
				excludes = append(excludes, strings.ToLower(cleanDirName(name.(string))))
			}
		case KwVars:
			specs := val.(map[string]interface{})
			for name, replace := range specs {
				replaceStr := replace.(string)
				name = strings.ToLower(strings.TrimSpace(name))

				// Wrap the key in hashes to make the processing faster later on
				variables["#"+name+"#"] = replaceStr
			}
		case KwSkipDirs:
			values := val.([]interface{})
			for _, dir := range values {
				dirStr := cleanDirName(dir.(string))
				skipDirs[strings.ToLower(dirStr)] = nil
			}
		case KwSave:
			specs := val.(map[string]interface{})
			for option, filename := range specs {
				switch option {
				case "missing":
					missingFileName = filename.(string)
				case "orphans":
					orphanFileName = filename.(string)
				case "log":
					logFileName = filename.(string)
				default:
					fmt.Fprintf(os.Stderr, "Invalid %s parameter '%s'\n", KwSave, option)
					passed = false
				}
			}
		default:
			return fmt.Errorf("Invalid JSON configuration keyword '%s'", key)
		}
	}

	// Get optional parameters
	if len(os.Args) > 2 {
		for index := 2; index < len(os.Args); index++ {
			crossRefNames = append(crossRefNames, os.Args[index])
		}
	}

	// Open output files
	if len(missingFileName) > 0 {
		missingWriter, err = os.Create(missingFileName)

		if err != nil {
			return err
		}
	}

	if len(orphanFileName) > 0 {
		orphanWriter, err = os.Create(orphanFileName)

		if err != nil {
			return err
		}
	}

	if len(logFileName) > 0 {
		logWriter, err = os.Create(logFileName)

		if err != nil {
			return err
		}
	}

	// Root dir is required
	if len(rootDir) == 0 {
		fmt.Fprintf(os.Stderr, "The root directory specification is required\n")
		passed = false
	} else {
		// Get the size of the root dir spec
		rootDirSize = len(rootDir)
	}

	// Add in subversion as a directory to skip
	skipDirs["/.svn"] = nil

	// CHeck for success
	if !passed {
		return fmt.Errorf("Errors were encountered processing the configuration")
	}

	// All done, return success
	return nil
}

func walkTree(path string, info os.FileInfo, err error) error {
	// Only process valid cfm or cfc file names and not directories
	if info.IsDir() {
		// Check for directoriess to skip
		relPath := path[rootDirSize:]

		if len(relPath) > 0 {
			// Clean and normalize the directory name
			relPath := cleanDirName(relPath)

			// See if its in the skip list
			_, exists := skipDirs[strings.ToLower(relPath)]

			if exists {
				// Skip the entire directory
				return filepath.SkipDir
			}

		}

		// Display a status if requested
		if verboseMode {
			fmt.Fprintf(logWriter, "Processing directory '%s'\n", path)
		}

		// DOn't process an actual directory
		return nil
	} else {
		// Processing a file but make sure it's one we want
		if !regexpFName.MatchString(filepath.Base(path)) {
			return nil
		}
	}

	// Process the file
	result := parseFile(path)

	if result != nil {
		fmt.Println(result)
	}

	// Return the result
	return result
}

// cleanDirName Clean up a directory name to a standard format for specifying a component name
// Any windows drive designation is removed
// All backslashes are changed to forward slashes
// A leading slash is dded is missing
// A trailing slash is removed is specified
//
// name The directory name to clean
func cleanDirName(name string) string {
	// Normalise slashes to flrward slashes
	name = strings.ReplaceAll(name, `\`, "/")

	// Remove any windows drive specification
	if len(name) > 2 && name[1] == ':' {
		name = name[2:]
	}

	// Add an initial slash if not there
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	// Remove any trailing slash
	if strings.HasSuffix(name, "/") {
		name = name[:len(name)-1]
	}

	// Return the clean name
	return name
}

func parseFile(fileName string) (err error) {
	// Skip this file if requested
	for _, skip := range excludes {
		if strings.HasSuffix(strings.ToLower(strings.ReplaceAll(fileName, `\`, "/")), skip) {
			return nil
		}
	}

	// Open and defer the closing of the file
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	defer file.Close()

	// Read each line and process
	lineNo := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(scanBuff, 5000000)

	for scanner.Scan() {
		// Process each line
		lineNo++

		/*
			 	Process the regular expression
				A 2 dimensional array is returned.
				0.1 is the name (either cffunction or cfinvoke)

				For cffunction
				1.3 Name keyword
				1.4 actual name for this function

				For cfinvoke
				1.3 Either "component" or "method"
				1.4 The component or method name
				2.3 The other keyword
				2.4 and its valu4
		*/
		currentLine = scanner.Text()
		matches := regExp.FindAllStringSubmatch(currentLine, -1)

		if len(matches) == 0 {
			continue
		}

		// Process cffunction or cfinvoke
		//fmt.Printf("Processing %s at line %d\n", fileName, lineNo)
		switch strings.ToLower(matches[0][1]) {
		case "cffunction":
			// Process the cffunction
			if err = processFunction(matches, fileName, lineNo); err != nil {
				return err
			}

		case "cfinvoke":
			// Save the invokes and process after all the functions have been built
			if err = deferInvoke(matches, fileName, lineNo); err != nil {
				return err
			}

		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Return status
	return err
}

// processFunction Initialize the function definition for processing by the invokes
// matches: results of the regular expression matching
// fileName: File name being processed for diagnostics
// lineNo: line number in that file the function definition occurred
// compFuncs: Component function information for that file
func processFunction(matches [][]string, fileName string, lineNo int) (err error) {
	// Get the component name from the file name and create a key name
	compName := removeSuffix(strings.ReplaceAll(fileName[rootDirSize:], "\\", "/"))
	mapName := strings.ToLower(compName)

	// Get the function definition for this file, creating one if not found
	componentDefinition, valid := xref[mapName]

	if !valid {
		componentDefinition = compDef{name: compName, funcs: make(map[string]funcDef)}
		xref[mapName] = componentDefinition
	}

	functionDefinitions := componentDefinition.funcs

	if len(matches) == 1 {
		fmt.Fprintf(logWriter, "Error for file %s at line %d\n", fileName, lineNo)
		fmt.Fprintf(logWriter, "Text:'%s'\n", currentLine)
		fmt.Println(currentLine)
	}

	// Make sure the 'name' keyword is defined
	if !strings.EqualFold("name", matches[1][3]) {
		return fmt.Errorf("The cffunction at line %d in file %s is invalid", lineNo, fileName)
	}

	// Process the name of this function
	funcName := matches[1][4]

	// Setup the function definition for this function
	functionDefinitions[strings.ToLower(funcName)] = funcDef{name: funcName, usedBy: make(map[string]funcUsage)}

	// Increment the number of functions
	totalFuncs++

	// Return result
	return err
}

func deferInvoke(matches [][]string, fileName string, lineNo int) (err error) {
	// Get and process the component and method parameters

	fileName = strings.ReplaceAll(fileName[rootDirSize:], `\`, "/")
	funcCall := defInvoke{fileName: fileName, line: lineNo}

	// Process keywords
	var component string
	var method string

	for index := range matches {
		if index == 0 {
			// Skip the first one
			continue
		}

		keyword := matches[index][3]
		value := matches[index][4]

		if strings.EqualFold("component", keyword) {
			component = strings.ReplaceAll(value, "\\", "/")
		} else if strings.EqualFold("method", keyword) {
			method = value
		} else {
			// Some custom invocations mave have "name" keywords, which we skip
			continue
		}
	}

	// The component may be missing if the function is in the same file
	if len(component) == 0 {
		// The component is 'this' file
		component = removeSuffix(fileName)
	}

	// Expand any variable specifications
	varInstances := findVars.FindAllString(component, -1)

	for _, spec := range varInstances {
		replacement, found := variables[strings.ToLower(spec)]

		if !found {
			fmt.Fprintf(logWriter, "The variable '%s' was not found", spec)
			return nil
		}

		component = strings.ReplaceAll(component, spec, replacement)
	}

	// Save the info for this invocation
	funcCall.component = component
	funcCall.method = method

	deferredList = append(deferredList, funcCall)
	return err
}

// removeSuffix Remove file suffix
func removeSuffix(name string) string {
	// See if there's a suffix
	dotLoc := strings.LastIndex(name, ".")
	if dotLoc > 0 {
		name = name[:dotLoc]
	}

	return name
}

// processInvoke Process the deferred invoke list
func processInvoke() {
	// Iterate through each invoke
	for _, spec := range deferredList {
		compInfo, err := lookupComponent(spec.component)

		if err != nil {
			missingFuncCt++
			fmt.Fprintf(missingWriter, "The component %s referenced in %s at line %d was not found\n",
				spec.component, spec.fileName, spec.line)
			continue
		}

		// Lookup the method in the function
		funcInfo, found := compInfo[strings.ToLower(spec.method)]

		if !found {
			missingMethodCt++
			fmt.Fprintf(missingWriter, "The method    %s referenced in %s at line %d was not found in function %s\n",
				spec.method, spec.fileName, spec.line, spec.component)
			continue
		}

		// Update the function info for the cross reference
		useKey := strings.ToLower(spec.fileName)
		usage, found := funcInfo.usedBy[useKey]

		if !found {
			// First reference for this file name
			usage = funcUsage{cleanName: spec.fileName, useLines: make([]int, 0, 5)}
		}

		// Update the use for this file
		usage.useLines = append(usage.useLines, spec.line)
		funcInfo.usedBy[useKey] = usage
	}
}

// lookupComponent Normalizes a component name and finds it in the xref
func lookupComponent(compName string) (map[string]funcDef, error) {
	// Convert to lower case firt
	key := strings.ToLower(compName)

	// Check for the native name first
	result, found := xref[key]

	if found {
		return result.funcs, nil
	}

	// Not there so try the tag paths
	for _, path := range tagPath {
		pathKey := path + "/" + key
		result, found = xref[pathKey]

		if found {
			return result.funcs, nil
		}
	}

	// No joy
	return nil, fmt.Errorf("The component '%s' was not found in the default path or in %q", compName, tagPath)
}

// processOrphans Funused components and functions
func processOrphans() {
	// Process each component in the crossref
	for _, component := range xref {
		// Process each function for this component
		for _, functions := range component.funcs {
			if len(functions.usedBy) == 0 {
				orphanList, found := orphans[component.name]

				// Allocate a new list
				if !found {
					orphanList = make([]string, 0)
					orphanCompCt++
				}

				// Add thi function to the list
				orphanList = append(orphanList, functions.name)
				orphanFuncCt++

				// Update the orphan function list for this component
				orphans[component.name] = orphanList
			}
		}
	}
}

// Display orphan information
func displayOrphans() {
	fmt.Fprintf(orphanWriter, "Orphaned functions and their component\n")
	for componentName, orphan := range orphans {
		fmt.Fprintf(orphanWriter, "Component: %s\n", componentName)
		for _, functionName := range orphan {
			fmt.Fprintf(orphanWriter, "    %s\n", functionName)
		}
	}
}

// Display cross reference data for the specified component
func crossReference(componentName string) {
	// Get the data for the component
	name := strings.ToLower(componentName)
	componentDef, found := xref[name]

	if !found {
		fmt.Fprintf(logWriter, "The component '%s' does not exist in the cross refernce data\n", componentName)
		fmt.Fprintf(os.Stderr, "The component '%s' does not exist in the cross refernce data\n", componentName)
		return
	}

	fmt.Fprintf(logWriter, "Cross reference for component: %s\n", componentName)

	// Iterate through all the functions
	for _, functionMap := range componentDef.funcs {
		fmt.Fprintf(logWriter, "    %s\n", functionMap.name)
		for _, usage := range functionMap.usedBy {
			fmt.Fprintf(logWriter, "            %s %v\n", usage.cleanName, usage.useLines)
		}
	}
}
