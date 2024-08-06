// Package dirs provides functions to sanitize directory and file names.
package dirs

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	sep      = string(os.PathSeparator)
	space    = " "
	spacep   = "%20"
	newline  = "\n"
	newlinep = "%0A"
)

var (
	home   = os.Getenv("HOME")
	pwd, _ = os.Getwd()
)

// UnExpand returns dir after expanding some directory shortcuts
//
//	$HOME -> ~
//
//	$PWD -> .
//
//	workdir -> /
func UnExpand(dir, workdir string) (outdir string) {
	if dir != "" {
		outdir = cleanDir(dir, pwd)
	}

	if workdir != "" {
		workdir = cleanDir(workdir, pwd)
		outdir = strings.Replace(outdir, workdir, "", 1)
	} else if home != pwd && pwd != "" {
		outdir = strings.Replace(outdir, pwd, ".", 1)
	}

	outdir = strings.Replace(outdir, home, "~", 1)

	outdir = PercentDecode(outdir)

	if outdir == "" {
		outdir = "/"
	}

	return
}

func PercentDecode(input string) (output string) {
	output = strings.ReplaceAll(input, spacep, space)
	output = strings.ReplaceAll(output, newlinep, newline)

	return
}

func PercentEncode(input string) (output string) {
	output = strings.ReplaceAll(input, space, spacep)
	output = strings.ReplaceAll(output, newline, newlinep)

	return
}

func cleanDir(dir, pwd string) (out string) {
	if strings.HasPrefix(dir, ".") {
		out = filepath.Clean(dir)
	} else if !strings.HasPrefix(dir, sep) {
		out = filepath.Join(pwd, dir)
	} else {
		out = dir
	}
	return
}
