package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/filter"
	"git.burning.moe/celediel/gt/internal/modes"
	"git.burning.moe/celediel/gt/internal/tables"
	"git.burning.moe/celediel/gt/internal/trash"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

const (
	appname    string = "gt"
	appdesc    string = "xdg trash cli"
	appversion string = "v0.0.1"
	yes        rune   = 'y'
	no         rune   = 'n'
)

var (
	loglvl         string
	f              *filter.Filter
	o, b, a, g, p  string
	ung, unp       string
	fo, do, ih     bool
	askconfirm     bool
	workdir, ogdir cli.Path
	recursive      bool
	termwidth      int
	termheight     int

	trashDir = filepath.Join(xdg.DataHome, "Trash")

	before_all = func(ctx *cli.Context) (err error) {
		// setup log
		log.SetReportTimestamp(true)
		log.SetTimeFormat(time.TimeOnly)
		if l, e := log.ParseLevel(loglvl); e == nil {
			log.SetLevel(l)
			// Some extra info for debug level
			if log.GetLevel() == log.DebugLevel {
				log.SetReportCaller(true)
			}
		}

		w, h, e := term.GetSize(int(os.Stdout.Fd()))
		if e != nil {
			w = 80
			h = 24
		}
		termwidth = w
		termheight = h

		return
	}

	before_commands = func(ctx *cli.Context) (err error) {
		// setup filter
		if f == nil {
			f, err = filter.New(o, b, a, g, p, ung, unp, fo, do, ih, ctx.Args().Slice()...)
		}
		log.Debugf("filter: %s", f.String())
		return
	}

	after = func(ctx *cli.Context) error {
		return nil
	}

	do_trash = &cli.Command{
		Name:    "trash",
		Aliases: []string{"tr"},
		Usage:   "Trash a file or files",
		Flags:   slices.Concat(trash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			fls, err := files.Find(workdir, recursive, f)
			if err != nil {
				return err
			}
			if len(fls) == 0 {
				fmt.Println("no files to trash")
				return nil
			}

			indices, err := tables.FilesTable(fls, termwidth, termheight, false, !f.Blank())
			if err != nil {
				return err
			}

			var selected files.Files
			for _, i := range indices {
				selected = append(selected, fls[i])
			}

			if len(selected) <= 0 {
				return nil
			}

			return confirm_trash(selected)
		},
	}

	// action launches interactive mode if run without args, or trashes files as args
	action = func(ctx *cli.Context) error {
		var (
			err error
		)

		if f == nil {
			f, err = filter.New(o, b, a, g, p, ung, unp, fo, do, ih)
		}
		if err != nil {
			return err
		}

		if len(ctx.Args().Slice()) != 0 {
			f.AddFileNames(ctx.Args().Slice()...)
			return do_trash.Action(ctx)
		} else {
			return interactive_mode()
		}
	}

	do_list = &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List trashed files",
		Flags:   slices.Concat(alreadyintrash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			log.Debugf("searching in directory %s for files", trashDir)

			// look for files
			fls, err := trash.FindFiles(trashDir, ogdir, f)

			var msg string
			if f.Blank() {
				msg = "trash is empty"
			} else {
				msg = "no files to show"
			}

			if len(fls) == 0 {
				fmt.Println(msg)
				return nil
			} else if err != nil {
				return err
			}

			// display them
			_, _, err = tables.InfoTable(fls, termwidth, termheight, true, false, modes.Listing)

			return err
		},
	}

	do_restore = &cli.Command{
		Name:    "restore",
		Aliases: []string{"re"},
		Usage:   "Restore a trashed file or files",
		Flags:   slices.Concat(alreadyintrash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			log.Debugf("searching in directory %s for files", trashDir)

			// look for files
			fls, err := trash.FindFiles(trashDir, ogdir, f)
			if len(fls) == 0 {
				fmt.Println("no files to restore")
				return nil
			} else if err != nil {
				return err
			}

			indices, _, err := tables.InfoTable(fls, termwidth, termheight, false, !f.Blank(), modes.Restoring)
			if err != nil {
				return err
			}

			var selected trash.Infos
			for _, i := range indices {
				selected = append(selected, fls[i])
			}

			if len(selected) <= 0 {
				return nil
			}

			return confirm_restore(selected)
		},
	}

	do_clean = &cli.Command{
		Name:    "clean",
		Aliases: []string{"cl"},
		Usage:   "Clean files from trash",
		Flags:   slices.Concat(alreadyintrash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			fls, err := trash.FindFiles(trashDir, ogdir, f)
			if len(fls) == 0 {
				fmt.Println("no files to clean")
				return nil
			} else if err != nil {
				return err
			}

			indices, _, err := tables.InfoTable(fls, termwidth, termheight, false, !f.Blank(), modes.Cleaning)
			if err != nil {
				return err
			}

			var selected trash.Infos
			for _, i := range indices {
				selected = append(selected, fls[i])
			}

			if len(selected) <= 0 {
				return nil
			}

			return confirm_clean(selected)
		},
	}

	global_flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "log",
			Usage:       "log level",
			Value:       "warn",
			Aliases:     []string{"l"},
			Destination: &loglvl,
		},
		&cli.BoolFlag{
			Name:               "confirm",
			Usage:              "ask for confirmation before executing any action",
			Value:              false,
			Aliases:            []string{"c"},
			DisableDefaultText: true,
			Destination:        &askconfirm,
		},
	}

	filter_flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "match",
			Usage:       "operate on files matching regex `PATTERN`",
			Aliases:     []string{"m"},
			Destination: &p,
		},
		&cli.StringFlag{
			Name:        "glob",
			Usage:       "operate on files matching `GLOB`",
			Aliases:     []string{"g"},
			Destination: &g,
		},
		&cli.StringFlag{
			Name:        "not-match",
			Usage:       "operate on files not matching regex `PATTERN`",
			Aliases:     []string{"M"},
			Destination: &unp,
		},
		&cli.StringFlag{
			Name:        "not-glob",
			Usage:       "operate on files not matching `GLOB`",
			Aliases:     []string{"G"},
			Destination: &ung,
		},
		&cli.StringFlag{
			Name:        "on",
			Usage:       "operate on files modified on `DATE`",
			Aliases:     []string{"o"},
			Destination: &o,
		},
		&cli.StringFlag{
			Name:        "after",
			Usage:       "operate on files modified before `DATE`",
			Aliases:     []string{"a"},
			Destination: &a,
		},
		&cli.StringFlag{
			Name:        "before",
			Usage:       "operate on files modified after `DATE`",
			Aliases:     []string{"b"},
			Destination: &b,
		},
		&cli.BoolFlag{
			Name:               "files-only",
			Usage:              "operate on files only",
			Aliases:            []string{"f"},
			DisableDefaultText: true,
			Destination:        &fo,
		},
		&cli.BoolFlag{
			Name:               "dirs-only",
			Usage:              "operate on directories only",
			Aliases:            []string{"d"},
			DisableDefaultText: true,
			Destination:        &do,
		},
		&cli.BoolFlag{
			Name:               "ignore-hidden",
			Usage:              "operate on unhidden files only",
			Aliases:            []string{"i"},
			DisableDefaultText: true,
			Destination:        &ih,
		},
	}

	trash_flags = []cli.Flag{
		&cli.BoolFlag{
			Name:               "recursive",
			Usage:              "operate on files recursively",
			Aliases:            []string{"r"},
			Destination:        &recursive,
			Value:              false,
			DisableDefaultText: true,
		},
		&cli.PathFlag{
			Name:        "work-dir",
			Usage:       "operate on files in this `DIRECTORY`",
			Aliases:     []string{"w"},
			Destination: &workdir,
		},
	}

	alreadyintrash_flags = []cli.Flag{
		&cli.PathFlag{
			Name:        "original-path",
			Usage:       "operate on files trashed from this `DIRECTORY`",
			Aliases:     []string{"O"},
			Destination: &ogdir,
		},
	}
)

func main() {
	app := &cli.App{
		Name:     appname,
		Usage:    appdesc,
		Version:  appversion,
		Before:   before_all,
		After:    after,
		Action:   action,
		Commands: []*cli.Command{do_trash, do_list, do_restore, do_clean},
		Flags:    global_flags,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func interactive_mode() error {
	var (
		fls      trash.Infos
		indicies []int
		mode     modes.Mode
		err      error
	)

	fls, err = trash.FindFiles(trashDir, ogdir, f)
	if err != nil {
		return err
	}

	if len(fls) <= 0 {
		log.Printf("no files to show")
		return nil
	}

	indicies, mode, err = tables.InfoTable(fls, termwidth, termheight, false, false, modes.Interactive)
	if err != nil {
		return err
	}

	var selected trash.Infos
	for _, i := range indicies {
		selected = append(selected, fls[i])
	}

	switch mode {
	case modes.Cleaning:
		for _, file := range selected {
			log.Debugf("gonna clean %s", file.Name())
		}
		if err := confirm_clean(selected); err != nil {
			return err
		}
	case modes.Restoring:
		for _, file := range selected {
			log.Debugf("gonna restore %s", file.Name())
		}
		if err := confirm_restore(selected); err != nil {
			return err
		}
	case modes.Interactive:
		return nil
	default:
		return fmt.Errorf("got bad mode %s", mode)
	}
	return nil
}

func confirm_restore(is trash.Infos) error {
	if confirm(fmt.Sprintf("restore %d selected files?", len(is))) {
		log.Info("doing the thing")
		restored, err := trash.Restore(is)
		if err != nil {
			return fmt.Errorf("restored %d files before error %s", restored, err)
		}
		fmt.Printf("restored %d files\n", restored)
	} else {
		fmt.Printf("not doing anything\n")
	}
	return nil
}

func confirm_clean(is trash.Infos) error {
	if confirm(fmt.Sprintf("remove %d selected files permanently from the trash?", len(is))) &&
		confirm(fmt.Sprintf("really remove all these %d selected files permanently from the trash forever??", len(is))) {
		log.Info("gonna remove some files forever")
		removed, err := trash.Remove(is)
		if err != nil {
			return fmt.Errorf("removed %d files before error %s", removed, err)
		}
		fmt.Printf("removed %d files\n", removed)
	} else {
		fmt.Printf("not doing anything\n")
	}
	return nil
}

func confirm_trash(fs files.Files) error {
	if confirm(fmt.Sprintf("trash %d selected files?", len(fs))) {
		tfs := make([]string, 0, len(fs))
		for _, file := range fs {
			log.Debugf("gonna trash %s", file.Filename())
			tfs = append(tfs, file.Filename())
		}

		trashed, err := trash.TrashFiles(trashDir, tfs...)
		if err != nil {
			return err
		}
		fmt.Printf("trashed %d files\n", trashed)
	} else {
		fmt.Printf("not doing anything\n")
		return nil
	}
	return nil
}

func confirm(prompt string) bool {
	if !askconfirm {
		return true
	}
	// TODO: handle errors better
	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
			panic(err)
		}
	}()

	fmt.Printf("%s [%s/%s]: ", prompt, string(yes), string(no))

	// read one byte from stdin
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		return false
	}

	return bytes.ToLower(b)[0] == 'y'
}
