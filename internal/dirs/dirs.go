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
	)

	outdir = filepath.Clean(dir)
	outdir = strings.ReplaceAll(outdir, home, "~")

	return
}
