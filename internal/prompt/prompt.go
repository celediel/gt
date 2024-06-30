package prompt

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"golang.org/x/term"
)

// TODO: use charm stuff for this

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

	fmt.Printf("%s [%s]: ", prompt, options)

	// read one byte from stdin
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		return 0
	}

	return bytes.ToLower(b)[0]
}