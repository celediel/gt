package files

import (
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
func (f DiskFile) Path() string    { return f.path }
func (f DiskFile) Date() time.Time { return f.modified }
func (f DiskFile) IsDir() bool     { return f.isdir }
func (f DiskFile) Filesize() int64 {
	if f.isdir {
		return -1
	}
	return f.filesize
}

func (f DiskFile) String() string {
	// this is unique enough because two files can't be named the same in the same directory
	// right???
	return f.name + f.path
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
		path:     filepath.Join(base_path, name),
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
		if f.Match(info) {
			files = append(files, DiskFile{
				path:     p,
				name:     name,
				filesize: info.Size(),
				modified: info.ModTime(),
				isdir:    info.IsDir(),
				mode:     info.Mode(),
			})
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

		if f.Match(info) {
			files = append(files, DiskFile{
				name:     name,
				path:     filepath.Join(path, name),
				modified: info.ModTime(),
				filesize: info.Size(),
				isdir:    info.IsDir(),
			})
		}
	}
	return
}
