// Package files finds and displays files on disk
package files

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.burning.moe/celediel/gt/internal/filter"
	"github.com/charmbracelet/log"
)

type File struct {
	name, path string
	filesize   int64
	modified   time.Time
	isdir      bool
}

type Files []File

func (f File) Name() string        { return f.name }
func (f File) Path() string        { return f.path }
func (f File) Filename() string    { return filepath.Join(f.path, f.name) }
func (f File) Modified() time.Time { return f.modified }
func (f File) Filesize() int64     { return f.filesize }
func (f File) IsDir() bool         { return f.isdir }

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

// is_in_recursive_dir checks `path` and parent directories
// of `path` up to `base` for a hidden parent
func is_in_recursive_dir(base, path string) bool {
	me := path
	for {
		me = filepath.Clean(me)
		if me == base {
			break
		}
		if strings.HasPrefix(filepath.Base(me), ".") {
			return true
		}
		me += string(os.PathSeparator) + ".."
	}
	return false
}

func walk_dir(dir string, f *filter.Filter) (files Files) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if dir == path {
			return nil
		}

		if is_in_recursive_dir(dir, path) && f.IgnoreHidden() {
			return nil
		}

		p, e := filepath.Abs(path)
		if e != nil {
			return err
		}

		name := d.Name()
		info, _ := d.Info()
		if f.Match(name, info.ModTime(), info.IsDir()) {
			log.Debugf("found matching file: %s %s", name, info.ModTime())
			i, err := os.Stat(p)
			if err != nil {
				log.Debugf("error in file stat: %s", err)
				return nil
			}
			files = append(files, File{
				path:     filepath.Dir(p),
				name:     name,
				filesize: i.Size(),
				modified: i.ModTime(),
				isdir:    i.IsDir(),
			})
		} else {
			log.Debugf("ignoring file %s (%s)", name, info.ModTime())
		}
		return nil
	})
	if err != nil {
		log.Errorf("error walking directory %s: %s", dir, err)
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

		if f.Match(name, info.ModTime(), info.IsDir()) {
			log.Debugf("found matching file: %s %s", name, info.ModTime())
			files = append(files, File{
				name:     name,
				path:     path,
				modified: info.ModTime(),
				filesize: info.Size(),
				isdir:    info.IsDir(),
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
