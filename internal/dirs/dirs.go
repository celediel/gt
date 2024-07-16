package dirs

import (
	"os"
	"path/filepath"
	"strings"
)

const sep = string(os.PathSeparator)

var (
	home   string = os.Getenv("HOME")
	pwd, _        = os.Getwd()
)

// UnExpand unexpands some directory shortcuts
//
// $HOME -> ~
// $PWD -> .
// workdir -> /
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

	outdir = UnEscape(outdir)

	if outdir == "" {
		outdir = "/"
	}

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

func UnEscape(input string) string {
	return strings.ReplaceAll(input, "%20", " ")
}
