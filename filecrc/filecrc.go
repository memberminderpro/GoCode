package main

import (
	"os"
)

func main() {
	rc := run(os.Args)

	os.Exit(rc)
}
