package filemode

import (
	"io/fs"
	"strconv"
)

// Parse parses a string of 3 or 4 numbers as a *NIX filesystem permission.
//
// "0777" or "777" -> fs.FileMode(0777)
//
// "0644" or "644" -> fs.FileMode(0644)
func Parse(in string) (fs.FileMode, error) {
	if in == "" {
		return fs.FileMode(0), nil
	}
	if len(in) == 3 {
		in = "0" + in
	}
	md, e := strconv.ParseUint(in, 8, 64)
	if e != nil {
		return 0, e
	}
	return fs.FileMode(md), nil
}
