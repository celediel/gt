package dirs

import (
	"os"
	"path/filepath"
	"strings"
)

// UnExpand unexpands some directory shortcuts
//
// $HOME -> ~
func UnExpand(dir string) (outdir string) {
	var (
		home = os.Getenv("HOME")
		pwd  string
		err  error
	)

	outdir = filepath.Clean(dir)

	pwd, err = os.Getwd()
	if err == nil && home != pwd {
		outdir = strings.Replace(outdir, pwd, "$PWD", 1)
	}

	outdir = strings.Replace(outdir, home, "~", 1)

	return
}
