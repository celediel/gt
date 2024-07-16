package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/filter"
	"git.burning.moe/celediel/gt/internal/prompt"
	"git.burning.moe/celediel/gt/internal/tables"
	"git.burning.moe/celediel/gt/internal/tables/modes"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

const (
	appname    string = "gt"
	appdesc    string = "xdg trash cli"
	appversion string = "v0.0.1"
)

var (
	loglvl           string
	f                *filter.Filter
	o, b, a, g, p, m string
	sm, lg           string
	ung, unp         string
	fo, do, sh, ni   bool
	askconfirm, all  bool
	workdir, ogdir   cli.Path
	recursive        bool
	termwidth        int
	termheight       int

	trashDir = filepath.Join(xdg.DataHome, "Trash")

	beforeAll = func(ctx *cli.Context) (err error) {
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

		// read the term height and width for tables
		w, h, e := term.GetSize(int(os.Stdout.Fd()))
		if e != nil {
			w = 80
			h = 24
		}
		termwidth = w
		termheight = h

		// ensure trash directories exist
		if _, e := os.Stat(trashDir); os.IsNotExist(e) {
			if err := os.Mkdir(trashDir, fs.FileMode(0755)); err != nil {
				return err
			}
		}
		if _, e := os.Stat(filepath.Join(trashDir, "info")); os.IsNotExist(e) {
			if err := os.Mkdir(filepath.Join(trashDir, "info"), fs.FileMode(0755)); err != nil {
				return err
			}
		}
		if _, e := os.Stat(filepath.Join(trashDir, "files")); os.IsNotExist(e) {
			if err := os.Mkdir(filepath.Join(trashDir, "files"), fs.FileMode(0755)); err != nil {
				return err
			}
		}

		return
	}

	// action launches interactive mode if run without args, or trashes files as args
	action = func(ctx *cli.Context) error {
		var (
			err error
		)

		if f == nil {
			md, e := parseMode(m)
			if e != nil {
				return e
			}
			f, err = filter.New(o, b, a, g, p, ung, unp, fo, do, false, sm, lg, md)
		}
		if err != nil {
			return err
		}

		if len(ctx.Args().Slice()) != 0 {
			var files_to_trash files.Files
			for _, arg := range ctx.Args().Slice() {
				file, e := files.NewDisk(arg)
				if e != nil {
					log.Fatalf("cannot trash '%s': No such file or directory", arg)
				}
				files_to_trash = append(files_to_trash, file)
			}
			return confirmTrash(askconfirm, files_to_trash)
		} else {
			return interactiveMode()
		}
	}

	beforeCommands = func(ctx *cli.Context) (err error) {
		// setup filter
		if f == nil {
			md, e := parseMode(m)
			if e != nil {
				return e
			}
			f, err = filter.New(o, b, a, g, p, ung, unp, fo, do, false, sm, lg, md, ctx.Args().Slice()...)
		}
		log.Debugf("filter: %s", f.String())
		return
	}

	beforeTrash = func(_ *cli.Context) (err error) {
		if f == nil {
			md, e := parseMode(m)
			if e != nil {
				return e
			}
			f, err = filter.New(o, b, a, g, p, ung, unp, fo, do, !sh, sm, lg, md)
		}
		log.Debugf("filter: %s", f.String())
		return
	}

	after = func(ctx *cli.Context) error {
		return nil
	}

	doTrash = &cli.Command{
		Name:    "trash",
		Aliases: []string{"tr"},
		Usage:   "Trash a file or files",
		Flags:   slices.Concat(trashFlags, filterFlags),
		Before:  beforeTrash,
		Action: func(ctx *cli.Context) error {
			var files_to_trash files.Files
			var selectall bool
			for _, arg := range ctx.Args().Slice() {
				file, e := files.NewDisk(arg)
				if e != nil {
					log.Debugf("%s wasn't really a file", arg)
					f.AddFileName(arg)
					continue
				}
				files_to_trash = append(files_to_trash, file)
				selectall = true
			}

			// if none of the args were files, then process find files based on filter
			if len(files_to_trash) == 0 {
				fls, err := files.FindDisk(workdir, recursive, f)
				if err != nil {
					return err
				}
				if len(fls) == 0 {
					fmt.Println("no files to trash")
					return nil
				}
				files_to_trash = append(files_to_trash, fls...)
				selectall = !f.Blank()
			}

			selected, _, err := tables.Select(files_to_trash, termwidth, termheight, false, selectall, false, workdir, modes.Trashing)
			if err != nil {
				return err
			}

			if len(selected) <= 0 {
				return nil
			}

			return confirmTrash(askconfirm, selected)
		},
	}

	doList = &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List trashed files",
		Flags:   slices.Concat(listFlags, alreadyintrashFlags, filterFlags),
		Before:  beforeCommands,
		Action: func(ctx *cli.Context) error {
			log.Debugf("searching in directory %s for files", trashDir)

			// look for files
			fls, err := files.FindTrash(trashDir, ogdir, f)

			var msg string
			log.Debugf("filter '%s' is blark? %t in %s", f, f.Blank(), ogdir)
			if f.Blank() && ogdir == "" {
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
			_, _, err = tables.Select(fls, termwidth, termheight, true, false, ni, workdir, modes.Listing)

			return err
		},
	}

	doRestore = &cli.Command{
		Name:    "restore",
		Aliases: []string{"re"},
		Usage:   "Restore a trashed file or files",
		Flags:   slices.Concat(cleanRestoreFlags, alreadyintrashFlags, filterFlags),
		Before:  beforeCommands,
		Action: func(ctx *cli.Context) error {
			log.Debugf("searching in directory %s for files", trashDir)

			// look for files
			fls, err := files.FindTrash(trashDir, ogdir, f)
			if len(fls) == 0 {
				fmt.Println("no files to restore")
				return nil
			} else if err != nil {
				return err
			}

			selected, _, err := tables.Select(fls, termwidth, termheight, false, all, all, workdir, modes.Restoring)
			if err != nil {
				return err
			}

			if len(selected) <= 0 {
				return nil
			}

			return confirmRestore(askconfirm || all, selected)
		},
	}

	doClean = &cli.Command{
		Name:    "clean",
		Aliases: []string{"cl"},
		Usage:   "Clean files from trash",
		Flags:   slices.Concat(cleanRestoreFlags, alreadyintrashFlags, filterFlags),
		Before:  beforeCommands,
		Action: func(ctx *cli.Context) error {
			fls, err := files.FindTrash(trashDir, ogdir, f)
			if len(fls) == 0 {
				fmt.Println("no files to clean")
				return nil
			} else if err != nil {
				return err
			}

			selected, _, err := tables.Select(fls, termwidth, termheight, false, all, all, workdir, modes.Cleaning)
			if err != nil {
				return err
			}

			if len(selected) <= 0 {
				return nil
			}

			return confirmClean(askconfirm, selected)
		},
	}

	globalFlags = []cli.Flag{
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

	filterFlags = []cli.Flag{
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
			Aliases:     []string{"O"},
			Destination: &o,
		},
		&cli.StringFlag{
			Name:        "after",
			Usage:       "operate on files modified before `DATE`",
			Aliases:     []string{"A"},
			Destination: &a,
		},
		&cli.StringFlag{
			Name:        "before",
			Usage:       "operate on files modified after `DATE`",
			Aliases:     []string{"B"},
			Destination: &b,
		},
		&cli.BoolFlag{
			Name:               "files-only",
			Usage:              "operate on files only",
			Aliases:            []string{"F"},
			DisableDefaultText: true,
			Destination:        &fo,
		},
		&cli.BoolFlag{
			Name:               "dirs-only",
			Usage:              "operate on directories only",
			Aliases:            []string{"D"},
			DisableDefaultText: true,
			Destination:        &do,
		},
		&cli.StringFlag{
			Name:        "min-size",
			Usage:       "operate on files larger than `SIZE`",
			Aliases:     []string{"N"},
			Destination: &sm,
		},
		&cli.StringFlag{
			Name:        "max-size",
			Usage:       "operate on files smaller than `SIZE`",
			Aliases:     []string{"X"},
			Destination: &lg,
		},
		&cli.StringFlag{
			Name:        "mode",
			Usage:       "operate on files matching mode `MODE`",
			Aliases:     []string{"x"},
			Destination: &m,
		},
	}

	trashFlags = []cli.Flag{
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
		&cli.BoolFlag{
			Name:               "hidden",
			Usage:              "operate on hidden files",
			Aliases:            []string{"H"},
			DisableDefaultText: true,
			Destination:        &sh,
		},
	}

	alreadyintrashFlags = []cli.Flag{
		&cli.PathFlag{
			Name:        "original-path",
			Usage:       "operate on files trashed from this `DIRECTORY`",
			Aliases:     []string{"o"},
			Destination: &ogdir,
		},
	}

	listFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "non-interactive",
			Usage:       "list files and quit",
			Aliases:     []string{"n"},
			Destination: &ni,
		},
	}

	cleanRestoreFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:               "all",
			Usage:              "operate on all files in trash",
			Aliases:            []string{"a"},
			Destination:        &all,
			DisableDefaultText: true,
		},
	}
)

func main() {
	app := &cli.App{
		Name:                   appname,
		Usage:                  appdesc,
		Version:                appversion,
		Before:                 beforeAll,
		After:                  after,
		Action:                 action,
		Commands:               []*cli.Command{doTrash, doList, doRestore, doClean},
		Flags:                  globalFlags,
		EnableBashCompletion:   true,
		UseShortOptionHandling: true,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func interactiveMode() error {
	var (
		infiles  files.Files
		selected files.Files
		mode     modes.Mode
		err      error
	)

	infiles, err = files.FindTrash(trashDir, ogdir, f)
	if err != nil {
		return err
	}

	if len(infiles) <= 0 {
		var msg string
		if f.Blank() {
			msg = "trash is empty"
		} else {
			msg = "no files to show"
		}
		fmt.Println(msg)
		return nil
	}

	selected, mode, err = tables.Select(infiles, termwidth, termheight, false, false, false, workdir, modes.Interactive)
	if err != nil {
		return err
	}

	switch mode {
	case modes.Cleaning:
		for _, file := range selected {
			log.Debugf("gonna clean %s", file.Name())
		}
		if err := confirmClean(askconfirm, selected); err != nil {
			return err
		}
	case modes.Restoring:
		for _, file := range selected {
			log.Debugf("gonna restore %s", file.Name())
		}
		if err := confirmRestore(askconfirm, selected); err != nil {
			return err
		}
	case modes.Interactive:
		return nil
	default:
		return fmt.Errorf("got bad mode %s", mode)
	}
	return nil
}

func confirmRestore(confirm bool, fs files.Files) error {
	if !confirm || prompt.YesNo(fmt.Sprintf("restore %d selected files?", len(fs))) {
		log.Info("doing the thing")
		restored, err := files.Restore(fs)
		if err != nil {
			return fmt.Errorf("restored %d files before error %s", restored, err)
		}
		fmt.Printf("restored %d files\n", restored)
	} else {
		fmt.Printf("not doing anything\n")
	}
	return nil
}

func confirmClean(confirm bool, fs files.Files) error {
	if prompt.YesNo(fmt.Sprintf("remove %d selected files permanently from the trash?", len(fs))) &&
		(!confirm || prompt.YesNo(fmt.Sprintf("really remove all these %d selected files permanently from the trash forever??", len(fs)))) {
		log.Info("gonna remove some files forever")
		removed, err := files.Remove(fs)
		if err != nil {
			return fmt.Errorf("removed %d files before error %s", removed, err)
		}
		fmt.Printf("removed %d files\n", removed)
	} else {
		fmt.Printf("not doing anything\n")
	}
	return nil
}

func confirmTrash(confirm bool, fs files.Files) error {
	if !confirm || prompt.YesNo(fmt.Sprintf("trash %d selected files?", len(fs))) {
		tfs := make([]string, 0, len(fs))
		for _, file := range fs {
			log.Debugf("gonna trash %s", file.Path())
			tfs = append(tfs, file.Path())
		}

		trashed, err := files.TrashFiles(trashDir, tfs...)
		if err != nil {
			return err
		}
		var f string
		if trashed == 1 {
			f = "file"
		} else {
			f = "files"
		}
		fmt.Printf("trashed %d %s\n", trashed, f)
	} else {
		fmt.Printf("not doing anything\n")
		return nil
	}
	return nil
}

func parseMode(in string) (fs.FileMode, error) {
	if in == "" {
		return fs.FileMode(0), nil
	}
	if len(m) == 3 {
		in = "0" + in
	}
	md, e := strconv.ParseUint(in, 8, 64)
	if e != nil {
		return 0, e
	}
	return fs.FileMode(md), nil
}
