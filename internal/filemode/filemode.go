// Package filemode does things io/fs doesn't do
package filemode

import (
	"io/fs"
	"strconv"
)

// Parse parses a string of 3 or 4 numbers as a *NIX filesystem permission.
//
//	"0777" or "777" -> fs.FileMode(0777)
//
//	"0644" or "644" -> fs.FileMode(0644)
func Parse(input string) (fs.FileMode, error) {
	const simplemodelen = 3
	if input == "" {
		return fs.FileMode(0), nil
	}
	if len(input) == simplemodelen {
		input = "0" + input
	}
	md, e := strconv.ParseUint(input, 8, 64)
	if e != nil {
		return 0, e
	}

	return fs.FileMode(md), nil
}
