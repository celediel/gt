package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/filter"
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
)

var (
	loglvl         string
	f              *filter.Filter
	o, b, a, g, p  string
	ung, unp       string
	workdir, ogdir cli.Path
	recursive      bool
	termwidth      int

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

		w, _, e := term.GetSize(int(os.Stdout.Fd()))
		if e != nil {
			w = 80
		}
		termwidth = w

		return
	}

	before_commands = func(ctx *cli.Context) (err error) {
		// setup filter
		if f == nil {
			f, err = filter.New(o, b, a, g, p, ung, unp, ctx.Args().Slice()...)
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
		Usage:   "trash a file or files",
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

			fls.Show(termwidth)
			if confirm(fmt.Sprintf("trash these %d files?", len(fls))) {
				tfs := make([]string, 0, len(fls))
				for _, file := range fls {
					log.Debugf("gonna trash %s", file.Filename())
					tfs = append(tfs, file.Filename())
				}

				trashed, err := trash.TrashFiles(trashDir, tfs...)
				if err != nil {
					return err
				}
				log.Printf("trashed %d files", trashed)
			} else {
				log.Info("not gonna do it")
				return nil
			}
			return nil
		},
	}

	do_list = &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "list trashed files",
		Flags:   slices.Concat(alreadyintrash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			log.Debugf("searching in directory %s for files", trashDir)

			// look for files
			files, err := trash.FindFiles(trashDir, ogdir, f)

			var msg string
			if f.Blank() {
				msg = "trash is empty"
			} else {
				msg = "no files to show"
			}

			if len(files) == 0 {
				fmt.Println(msg)
				return nil
			} else if err != nil {
				return err
			}

			// display them
			files.Show(termwidth)

			return nil
		},
	}

	do_restore = &cli.Command{
		Name:    "restore",
		Aliases: []string{"re"},
		Usage:   "restore a trashed file or files",
		Flags:   slices.Concat(alreadyintrash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			log.Debugf("searching in directory %s for files", trashDir)

			// look for files
			files, err := trash.FindFiles(trashDir, ogdir, f)
			if len(files) == 0 {
				fmt.Println("no files to restore")
				return nil
			} else if err != nil {
				return err
			}

			files.Show(termwidth)
			if confirm(fmt.Sprintf("restore these %d files?", len(files))) {
				log.Info("doing the thing")
				restored, err := trash.Restore(files)
				if err != nil {
					return fmt.Errorf("restored %d files before error %s", restored, err)
				}
				log.Printf("restored %d files\n", restored)
			} else {
				log.Info("not gonna do it")
			}

			return nil
		},
	}

	do_clean = &cli.Command{
		Name:    "clean",
		Aliases: []string{"cl"},
		Usage:   "clean files from trash",
		Flags:   slices.Concat(alreadyintrash_flags, filter_flags),
		Before:  before_commands,
		Action: func(ctx *cli.Context) error {
			files, err := trash.FindFiles(trashDir, ogdir, f)
			if len(files) == 0 {
				fmt.Println("no files to clean")
				return nil
			} else if err != nil {
				return err
			}

			files.Show(termwidth)
			if confirm(fmt.Sprintf("remove these %d files permanently from the trash?", len(files))) &&
				confirm(fmt.Sprintf("really remove all %d of these files permanently from the trash forever??", len(files))) {
				log.Info("gonna remove some files forever")
				removed, err := trash.Remove(files)
				if err != nil {
					return fmt.Errorf("removed %d files before error %s", removed, err)
				}
				log.Printf("removed %d files\n", removed)
			} else {
				log.Printf("left %d files alone", len(files))
			}
			return nil
		},
	}

	global_flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "log",
			Usage:       "Log level",
			Value:       "warn",
			Aliases:     []string{"l"},
			Destination: &loglvl,
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
	}

	trash_flags = []cli.Flag{
		&cli.BoolFlag{
			Name:               "recursive",
			Usage:              "trash files recursively",
			Aliases:            []string{"r"},
			Destination:        &recursive,
			Value:              false,
			DisableDefaultText: true,
		},
		&cli.PathFlag{
			Name:        "work-dir",
			Usage:       "trash files in this `DIRECTORY`",
			Aliases:     []string{"w"},
			Destination: &workdir,
		},
	}

	alreadyintrash_flags = []cli.Flag{
		&cli.PathFlag{
			Name:        "original-path",
			Usage:       "restore files trashed from this `DIRECTORY`",
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
		Commands: []*cli.Command{do_trash, do_list, do_restore, do_clean},
		Flags:    global_flags,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func confirm(s string) bool {
	r := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/n]: ", s)
	got, err := r.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if len(got) < 2 {
		return false
	} else {
		return strings.ToLower(strings.TrimSpace(got))[0] == 'y'
	}
}
