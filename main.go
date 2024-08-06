// Package main does the thing
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"git.burning.moe/celediel/gt/internal/filemode"
	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/filter"
	"git.burning.moe/celediel/gt/internal/interactive"
	"git.burning.moe/celediel/gt/internal/interactive/modes"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
)

const (
	appname     string = "gt"
	appsubtitle string = "xdg trash cli"
	appversion  string = "v0.0.2"
	appdesc     string = `A small command line program to interface with the
Freedesktop.org / XDG trash specification.

Run with no command or filename(s) to start interactive mode.

See gt(1) for more information.`
	executePerm = fs.FileMode(0755)
)

var (
	fltr                       *filter.Filter
	loglvl                     string
	onArg, beforeArg, afterArg string
	globArg, patternArg        string
	unGlobArg, unPatternArg    string
	modeArg, minArg, maxArg    string
	filesOnlyArg, dirsOnlyArg  bool
	hiddenArg, noInterArg      bool
	askconfirm, all            bool
	workdir, ogdir             cli.Path
	recursive                  bool

	beforeAll = func(_ *cli.Context) error {
		// setup log
		log.SetReportTimestamp(true)
		log.SetTimeFormat(time.TimeOnly)
		if level, err := log.ParseLevel(loglvl); err == nil {
			log.SetLevel(level)
			// Some extra info for debug level
			if log.GetLevel() == log.DebugLevel {
				log.SetReportCaller(true)
			}
		} else {
			log.Errorf("unknown log level '%s' (possible values: debug, info, warn, error, fatal, default: warn)", loglvl)
		}

		// ensure personal trash directories exist
		homeTrash := filepath.Join(xdg.DataHome, "Trash")
		if _, e := os.Stat(filepath.Join(homeTrash, "info")); os.IsNotExist(e) {
			if err := os.MkdirAll(filepath.Join(homeTrash, "info"), executePerm); err != nil {
				return err
			}
		}
		if _, e := os.Stat(filepath.Join(homeTrash, "files")); os.IsNotExist(e) {
			if err := os.MkdirAll(filepath.Join(homeTrash, "files"), executePerm); err != nil {
				return err
			}
		}

		return nil
	}

	// action launches interactive mode if run without args, or trashes files as args.
	action = func(ctx *cli.Context) error {
		var (
			err error
		)

		if fltr == nil {
			md, e := filemode.Parse(modeArg)
			if e != nil {
				return e
			}
			fltr, err = filter.New(onArg, beforeArg, afterArg, globArg, patternArg, unGlobArg, unPatternArg, filesOnlyArg, dirsOnlyArg, false, minArg, maxArg, md)
		}
		if err != nil {
			return err
		}

		if len(ctx.Args().Slice()) == 0 {
			// no ags, so do interactive mode
			var (
				infiles  files.Files
				selected files.Files
				mode     modes.Mode
				err      error
			)

			infiles = files.FindInAllTrashes(ogdir, fltr)
			if len(infiles) <= 0 {
				var msg string
				if fltr.Blank() {
					msg = "trash is empty"
				} else {
					msg = "no files to show"
				}
				fmt.Fprintln(os.Stdout, msg)
				return nil
			}
			selected, mode, err = interactive.Select(infiles, false, false, workdir, modes.Interactive)
			if err != nil {
				return err
			}
			switch mode {
			case modes.Cleaning:
				for _, file := range selected {
					log.Debugf("gonna clean %s", file.Name())
				}
				if err := files.ConfirmClean(askconfirm, selected); err != nil {
					return err
				}
			case modes.Restoring:
				for _, file := range selected {
					log.Debugf("gonna restore %s", file.Name())
				}
				if err := files.ConfirmRestore(askconfirm, selected); err != nil {
					return err
				}
			case modes.Interactive:
				return nil
			default:
				return fmt.Errorf("got bad mode %s", mode)
			}
			return nil
		}

		// args, so try to trash files
		var filesToTrash files.Files
		for _, arg := range ctx.Args().Slice() {
			file, e := files.NewDisk(arg)
			if e != nil {
				log.Errorf("cannot trash '%s': No such file or directory", arg)
				continue
			}
			filesToTrash = append(filesToTrash, file)
		}
		return files.ConfirmTrash(askconfirm, filesToTrash)
	}

	beforeCommands = func(ctx *cli.Context) (err error) {
		// setup filter
		if fltr == nil {
			md, e := filemode.Parse(modeArg)
			if e != nil {
				return e
			}
			fltr, err = filter.New(onArg, beforeArg, afterArg, globArg, patternArg, unGlobArg, unPatternArg, filesOnlyArg, dirsOnlyArg, false, minArg, maxArg, md, ctx.Args().Slice()...)
		}
		log.Debugf("filter: %s", fltr.String())
		return
	}

	beforeTrash = func(_ *cli.Context) (err error) {
		if fltr == nil {
			md, e := filemode.Parse(modeArg)
			if e != nil {
				return e
			}
			fltr, err = filter.New(onArg, beforeArg, afterArg, globArg, patternArg, unGlobArg, unPatternArg, filesOnlyArg, dirsOnlyArg, !hiddenArg, minArg, maxArg, md)
		}
		log.Debugf("filter: %s", fltr.String())
		return
	}

	after = func(_ *cli.Context) error {
		return nil
	}

	doTrash = &cli.Command{
		Name:      "trash",
		Aliases:   []string{"tr"},
		Usage:     "Trash a file or files",
		UsageText: "[command options] [filename(s)]",
		Flags:     slices.Concat(trashingFlags, filterFlags),
		Before:    beforeTrash,
		Action: func(ctx *cli.Context) error {
			var filesToTrash files.Files
			for _, arg := range ctx.Args().Slice() {
				file, e := files.NewDisk(arg)
				if e != nil || workdir != "" {
					log.Debugf("%s wasn't really a file", arg)
					fltr.AddFileName(arg)
					continue
				}
				filesToTrash = append(filesToTrash, file)
			}

			// if none of the args were files, then find files based on filter
			if len(filesToTrash) == 0 {
				fls := files.FindDisk(workdir, recursive, fltr)
				if len(fls) == 0 {
					fmt.Fprintln(os.Stdout, "no files to trash")
					return nil
				}
				filesToTrash = append(filesToTrash, fls...)
			}

			selected, _, err := interactive.Select(filesToTrash, false, false, workdir, modes.Trashing)
			if err != nil {
				return err
			}

			if len(selected) <= 0 {
				return nil
			}

			return files.ConfirmTrash(askconfirm, selected)
		},
	}

	doList = &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List trashed files",
		Flags:   slices.Concat(listFlags, trashedFlags, filterFlags),
		Before:  beforeCommands,
		Action: func(_ *cli.Context) error {
			fls := files.FindInAllTrashes(ogdir, fltr)

			var msg string
			log.Debugf("filter '%s' is blank? %t in %s", fltr, fltr.Blank(), ogdir)
			if fltr.Blank() && ogdir == "" {
				msg = "trash is empty"
			} else {
				msg = "no files to show"
			}

			if len(fls) == 0 {
				fmt.Fprintln(os.Stdout, msg)
				return nil
			}

			return interactive.Show(fls, noInterArg, workdir)
		},
	}

	doRestore = &cli.Command{
		Name:      "restore",
		Aliases:   []string{"re"},
		Usage:     "Restore a trashed file or files",
		UsageText: "[command options] [filename(s)]",
		Flags:     slices.Concat(cleanRestoreFlags, trashedFlags, filterFlags),
		Before:    beforeCommands,
		Action: func(_ *cli.Context) error {
			fls := files.FindInAllTrashes(ogdir, fltr)
			if len(fls) == 0 {
				fmt.Fprintln(os.Stdout, "no files to restore")
				return nil
			}

			selected, _, err := interactive.Select(fls, all, all, workdir, modes.Restoring)
			if err != nil {
				return err
			}

			if len(selected) <= 0 {
				return nil
			}

			return files.ConfirmRestore(askconfirm || all, selected)
		},
	}

	doClean = &cli.Command{
		Name:      "clean",
		Aliases:   []string{"cl"},
		Usage:     "Clean files from trash",
		UsageText: "[command options] [filename(s)]",
		Flags:     slices.Concat(cleanRestoreFlags, trashedFlags, filterFlags),
		Before:    beforeCommands,
		Action: func(_ *cli.Context) error {
			fls := files.FindInAllTrashes(ogdir, fltr)
			if len(fls) == 0 {
				fmt.Fprintln(os.Stdout, "no files to clean")
				return nil
			}

			selected, _, err := interactive.Select(fls, all, all, workdir, modes.Cleaning)
			if err != nil {
				return err
			}

			if len(selected) <= 0 {
				return nil
			}

			return files.ConfirmClean(askconfirm, selected)
		},
	}

	globalFlags = []cli.Flag{
		&cli.StringFlag{
			Name:        "log",
			Usage:       "set log level",
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
			Destination: &patternArg,
		},
		&cli.StringFlag{
			Name:        "glob",
			Usage:       "operate on files matching `GLOB`",
			Aliases:     []string{"g"},
			Destination: &globArg,
		},
		&cli.StringFlag{
			Name:        "not-match",
			Usage:       "operate on files not matching regex `PATTERN`",
			Aliases:     []string{"M"},
			Destination: &unPatternArg,
		},
		&cli.StringFlag{
			Name:        "not-glob",
			Usage:       "operate on files not matching `GLOB`",
			Aliases:     []string{"G"},
			Destination: &unGlobArg,
		},
		&cli.StringFlag{
			Name:        "on",
			Usage:       "operate on files modified on `DATE`",
			Aliases:     []string{"O"},
			Destination: &onArg,
		},
		&cli.StringFlag{
			Name:        "after",
			Usage:       "operate on files modified before `DATE`",
			Aliases:     []string{"A"},
			Destination: &afterArg,
		},
		&cli.StringFlag{
			Name:        "before",
			Usage:       "operate on files modified after `DATE`",
			Aliases:     []string{"B"},
			Destination: &beforeArg,
		},
		&cli.BoolFlag{
			Name:               "files-only",
			Usage:              "operate on files only",
			Aliases:            []string{"F"},
			DisableDefaultText: true,
			Destination:        &filesOnlyArg,
		},
		&cli.BoolFlag{
			Name:               "dirs-only",
			Usage:              "operate on directories only",
			Aliases:            []string{"D"},
			DisableDefaultText: true,
			Destination:        &dirsOnlyArg,
		},
		&cli.StringFlag{
			Name:        "min-size",
			Usage:       "operate on files larger than `SIZE`",
			Aliases:     []string{"N"},
			Destination: &minArg,
		},
		&cli.StringFlag{
			Name:        "max-size",
			Usage:       "operate on files smaller than `SIZE`",
			Aliases:     []string{"X"},
			Destination: &maxArg,
		},
		&cli.StringFlag{
			Name:        "mode",
			Usage:       "operate on files matching mode `MODE`",
			Aliases:     []string{"x"},
			Destination: &modeArg,
		},
	}

	trashingFlags = []cli.Flag{
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
			Destination:        &hiddenArg,
		},
	}

	trashedFlags = []cli.Flag{
		&cli.PathFlag{
			Name:        "original-path",
			Usage:       "operate on files trashed from this `DIRECTORY`",
			Aliases:     []string{"o"},
			Destination: &ogdir,
		},
	}

	listFlags = []cli.Flag{
		&cli.BoolFlag{
			Name:               "non-interactive",
			Usage:              "list files and quit",
			Aliases:            []string{"n"},
			Destination:        &noInterArg,
			DisableDefaultText: true,
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
		Usage:                  appsubtitle,
		Version:                appversion,
		Before:                 beforeAll,
		After:                  after,
		Action:                 action,
		Commands:               []*cli.Command{doTrash, doList, doRestore, doClean},
		Flags:                  globalFlags,
		UsageText:              appname + " [global options] [command [command options] / filename(s)]",
		Description:            appdesc,
		EnableBashCompletion:   true,
		UseShortOptionHandling: true,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
