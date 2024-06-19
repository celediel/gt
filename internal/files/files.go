// Package files finds and displays files on disk
package files

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"git.burning.moe/celediel/gt/internal/filter"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
)

type File struct {
	name, path string
	filesize   int64
	modified   time.Time
}

type Files []File

func (f File) Name() string        { return f.name }
func (f File) Path() string        { return f.path }
func (f File) Filename() string    { return filepath.Join(f.path, f.name) }
func (f File) Modified() time.Time { return f.modified }
func (f File) Filesize() int64     { return f.filesize }

func (fls Files) Table(width int) string {
	// sort newest on top
	slices.SortStableFunc(fls, SortByModifiedReverse)

	data := [][]string{}
	for _, file := range fls {
		t := humanize.Time(file.modified)
		b := humanize.Bytes(uint64(file.filesize))
		data = append(data, []string{
			file.name,
			file.path,
			t,
			b,
		})
	}
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		Width(width).
		Headers("filename", "path", "modified", "size").
		Rows(data...)

	return fmt.Sprint(t)
}

func (fls Files) Show(width int) {
	fmt.Println(fls.Table(width))
}

func Find(dir string, recursive bool, f *filter.Filter) (files Files, err error) {
	if dir == "." || dir == "" {
		var d string
		if d, err = os.Getwd(); err != nil {
			return
		} else {
			dir = d
		}
	}

	var recursively string
	if recursive {
		recursively = " recursively"
	}

	log.Debugf("gonna find files%s in %s matching %s", recursively, dir, f)

	if recursive {
		files = append(files, walk_dir(dir, f)...)
	} else {
		files = append(files, read_dir(dir, f)...)
	}

	return
}

func walk_dir(dir string, f *filter.Filter) (files Files) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		p, e := filepath.Abs(path)
		if e != nil {
			return err
		}
		name := d.Name()
		info, _ := d.Info()
		if f.Match(name, info.ModTime()) {
			log.Debugf("found matching file: %s %s", name, info.ModTime())
			i, _ := os.Stat(p)
			files = append(files, File{
				path:     filepath.Dir(p),
				name:     name,
				filesize: i.Size(),
				modified: i.ModTime(),
			})
		} else {
			log.Debugf("ignoring file %s (%s)", name, info.ModTime())
		}
		return nil
	})
	if err != nil {
		return []File{}
	}
	return
}

func read_dir(dir string, f *filter.Filter) (files Files) {
	fs, err := os.ReadDir(dir)
	if err != nil {
		return []File{}
	}
	for _, file := range fs {
		name := file.Name()

		if name == dir {
			continue
		}

		info, err := file.Info()
		if err != nil {
			return []File{}
		}

		path := filepath.Dir(filepath.Join(dir, name))

		if f.Match(name, info.ModTime()) {
			log.Debugf("found matching file: %s %s", name, info.ModTime())
			files = append(files, File{
				name:     name,
				path:     path,
				modified: info.ModTime(),
				filesize: info.Size(),
			})
		} else {
			log.Debugf("ignoring file %s (%s)", name, info.ModTime())
		}
	}
	return
}

func SortByModified(a, b File) int {
	if a.modified.After(b.modified) {
		return 1
	} else if a.modified.Before(b.modified) {
		return -1
	} else {
		return 0
	}
}

func SortByModifiedReverse(a, b File) int {
	if a.modified.Before(b.modified) {
		return 1
	} else if a.modified.After(b.modified) {
		return -1
	} else {
		return 0
	}
}

func SortBySize(a, b File) int {
	if a.filesize > b.filesize {
		return 1
	} else if a.filesize < b.filesize {
		return -1
	} else {
		return 0
	}
}

func SortBySizeReverse(a, b File) int {
	if a.filesize < b.filesize {
		return 1
	} else if a.filesize > b.filesize {
		return -1
	} else {
		return 0
	}
}
