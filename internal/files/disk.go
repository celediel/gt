package files

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.burning.moe/celediel/gt/internal/filter"

	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
)

type DiskFile struct {
	name, path string
	filesize   int64
	modified   time.Time
	isdir      bool
}

func (f DiskFile) Name() string    { return f.name }
func (f DiskFile) Path() string    { return filepath.Join(f.path, f.name) }
func (f DiskFile) Date() time.Time { return f.modified }
func (f DiskFile) IsDir() bool     { return f.isdir }
func (f DiskFile) Filesize() int64 {
	if f.isdir {
		return -1
	}
	return f.filesize
}

func (f DiskFile) String() string {
	return fmt.Sprintf(string_format, f.name, f.path, f.modified.Format(time.UnixDate), f.filesize, f.isdir)
}

func NewDisk(path string) (DiskFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return DiskFile{}, err
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		log.Errorf("couldn't get absolute path for %s", path)
		abs = path
	}

	name := filepath.Base(abs)
	base_path := filepath.Dir(abs)

	log.Debugf("%s (base:%s) (size:%s) (modified:%s) exists",
		name, base_path, humanize.Bytes(uint64(info.Size())), info.ModTime())

	return DiskFile{
		name:     name,
		path:     base_path,
		filesize: info.Size(),
		modified: info.ModTime(),
		isdir:    info.IsDir(),
	}, nil
}

func FindDisk(dir string, recursive bool, f *filter.Filter) (files Files, err error) {
	dir = filepath.Clean(dir)
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

// is_in_hidden_dir checks `path` and parent directories
// of `path` up to `base` for a hidden parent
func is_in_hidden_dir(base, path string) bool {
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

		if is_in_hidden_dir(dir, path) && f.IgnoreHidden() {
			return nil
		}

		p, e := filepath.Abs(path)
		if e != nil {
			return err
		}

		name := d.Name()
		info, _ := d.Info()
		if f.Match(name, info.ModTime(), info.Size(), info.IsDir()) {
			log.Debugf("found matching file: %s %s", name, info.ModTime())
			i, err := os.Stat(p)
			if err != nil {
				log.Debugf("error in file stat: %s", err)
				return nil
			}
			files = append(files, DiskFile{
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
		return Files{}
	}
	return
}

func read_dir(dir string, f *filter.Filter) (files Files) {
	fs, err := os.ReadDir(dir)
	if err != nil {
		return Files{}
	}
	for _, file := range fs {
		name := file.Name()

		if name == dir {
			continue
		}

		info, err := file.Info()
		if err != nil {
			return Files{}
		}

		path := filepath.Dir(filepath.Join(dir, name))

		if f.Match(name, info.ModTime(), info.Size(), info.IsDir()) {
			log.Debugf("found matching file: %s %s", name, info.ModTime())
			files = append(files, DiskFile{
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
