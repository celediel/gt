package dirs

import (
	"os"
	"path/filepath"
	"strings"
)

// UnExpand unexpands some directory shortcuts
//
// $HOME -> ~
// $PWD -> .
// workdir -> .
func UnExpand(dir, workdir string) (outdir string) {
	outdir = filepath.Clean(dir)
	home := os.Getenv("HOME")

	if pwd, err := os.Getwd(); err == nil && home != pwd {
		outdir = strings.Replace(outdir, pwd, ".", 1)
	} else if workdir != "" {
		outdir = strings.Replace(outdir, workdir, "", 1)
	}

	outdir = strings.Replace(outdir, home, "~", 1)

	outdir = UnEscape(outdir)

	if outdir == "" {
		outdir = "/"
	}

	return
}

func UnEscape(input string) string {
	return strings.ReplaceAll(input, "%20", " ")
}
