package main

import (
	"testing"
)

func TestMain(m *testing.M) {
	args := []string{"main", "config.json"}

	run(args)
}
