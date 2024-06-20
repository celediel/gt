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
	var (
		home = os.Getenv("HOME")
		pwd  string
		err  error
	)

	outdir = filepath.Clean(dir)

	if workdir != "" {
		outdir = strings.Replace(outdir, workdir, ".", 1)
	}

	pwd, err = os.Getwd()
	if err == nil && home != pwd {
		outdir = strings.Replace(outdir, pwd, ".", 1)
	}

	outdir = strings.Replace(outdir, home, "~", 1)

	return
}
