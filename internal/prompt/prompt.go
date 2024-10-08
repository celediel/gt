// Package prompt implements prompt functions for runes and strings.
package prompt

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

func YesNo(prompt string) bool {
	return AskRune(prompt, "y/n") == 'y'
}

func AskRune(prompt, options string) byte {
	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Fprintf(os.Stdout, "%s [%s]: ", prompt, options)

	// read one byte from stdin
	one := make([]byte, 1)
	_, err = os.Stdin.Read(one)
	if err != nil {
		return 0
	}

	return bytes.ToLower(one)[0]
}

func NewPath(path string) (string, bool) {
	for {
		answer := AskRune(fmt.Sprintf("file %s exists, overwrite, rename, or cancel?", path), "o/r/c")
		switch answer {
		case 'o', 'O':
			return path, false
		case 'r', 'R':
			if err := huh.NewInput().
				Title("input a new filename").
				Value(&path).
				Run(); err != nil {
				return path, false
			}
			if _, e := os.Lstat(path); e != nil {
				return path, false
			}
		default:
			return path, true
		}
	}
}
