package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	a := assert.New(t)
	args := []string{"main", "-c", "config.json"}

	rc := run(args)

	a.Equal(rc, 0)
}
