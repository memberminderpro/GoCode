package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pborman/getopt/v2"
)

const (
	flagRetainCt = 'r' // Flag for number of files to retain
	flagForceAll = 'f' // Flag to indicate that you want to delete all
)

var (
	parmRetainCt int    = 0     // Number of files to retain
	parmForceAll bool   = false // It's OK to delete all
	parmBaseName string = ""    // Base name of the zip file
)

func main() {
	// Need some args
	if err := getParms(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		usage()
		os.Exit(2)
	}

	// Process
	rc := process()

	os.Exit(rc)

}

func getParms(args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("you must specify configuration parameters")
	}

	// Setup for processing
	flagSet := getopt.New()

	// Setup specs
	flagSet.Flag(&parmRetainCt, flagRetainCt, "Number of files to retain (must be greter than zero)")
	flagSet.Flag(&parmForceAll, flagForceAll, "Flag to indicate a force to allow a retain count of zero")

	// Parse the arguments
	err := flagSet.Getopt(args, nil)

	if err != nil {
		return err
	}

	// Get the input file name
	posArgs := flagSet.Args()

	if len(posArgs) != 1 {
		return fmt.Errorf("you must specify an input file name")
	}

	parmBaseName = posArgs[0]

	// Quick checks
	status := true
	if parmRetainCt <= 0 {
		if parmRetainCt == 0 {
			if !parmForceAll {
				fmt.Fprintf(os.Stderr, "You must specify the '%c' flag for a retain count of zero\n", flagForceAll)
				status = false
			}
		} else {
			fmt.Fprintf(os.Stderr, "The retain count value of %d is invalid\n", parmRetainCt)
			status = false
		}
	} else {
		if parmForceAll {
			fmt.Fprintf(os.Stderr, "The force flag, '%c', can only be used with a retain count of zero\n", flagForceAll)
			status = false
		}
	}

	// Return the values
	if !status {
		return fmt.Errorf("processing terminated due to errors")
	}

	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, " Usage: [-%c] -%c count baseFileName\n", flagForceAll, flagRetainCt)
	fmt.Fprintf(os.Stderr, "  %c: The number of zip files to retain (required)\n", flagRetainCt)
	fmt.Fprintf(os.Stderr, "  %c: Allow a retain cound of zero (required if retain count is zero)\n", flagForceAll)
	fmt.Fprintf(os.Stderr, "\n")
}

func process() int {
	// Cleanup first
	fileName := strings.ReplaceAll(parmBaseName, `\`, "/")

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
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 2
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

	// Buid a map of files by suffix
	fileMap := make(map[int]string)
	numList := make([]int, 0)

	// Collect all the files, by suffix
	nameRegEx := regexp.MustCompile("^" + baseName + "([0-9]+)" + suffix + "$")

	for _, info := range fileList {
		if info.IsDir() {
			continue
		}

		matches := nameRegEx.FindAllStringSubmatch(info.Name(), -1)
		if len(matches) > 0 {
			num, _ := strconv.Atoi(matches[0][1])

			// Add the value to the map
			fileMap[num] = info.Name()
			numList = append(numList, num)
		}
	}

	// Check
	if len(numList) == 0 {
		fmt.Fprintf(os.Stderr, "There are no files found to cleanup\n")
		return 2
	}

	// Sort the number list
	sort.Ints(numList)

	// Starting at the start, go until you hit the retain count
	lastDeleted := len(numList) - parmRetainCt
	for index := 0; index < lastDeleted; index++ {
		// Get the name and delete the file
		name := fileMap[numList[index]]
		err := os.Remove(name)

		if err == nil {
			fmt.Fprintf(os.Stdout, "Deleted %s\n", name)
		} else {
			fmt.Fprintf(os.Stderr, "Error deleteing '%s': %s\n", name, err)
			return 2
		}
	}

	// Return success
	return 0
}
