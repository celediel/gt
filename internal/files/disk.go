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
	mode       fs.FileMode
}

func (f DiskFile) Name() string      { return f.name }
func (f DiskFile) Path() string      { return f.path }
func (f DiskFile) Date() time.Time   { return f.modified }
func (f DiskFile) IsDir() bool       { return f.isdir }
func (f DiskFile) Mode() fs.FileMode { return f.mode }
func (f DiskFile) Filesize() int64 {
	if f.isdir {
		return 0
	}
	return f.filesize
}

func (f DiskFile) String() string {
	// this is unique enough because two files can't be named the same in the same directory
	// right???
	return f.name + f.path
}

func NewDisk(path string) (DiskFile, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return DiskFile{}, err
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		log.Errorf("couldn't get absolute path for %s", path)
		abs = path
	}

	name := filepath.Base(abs)
	basePath := filepath.Dir(abs)

	log.Debugf("%s (base:%s) (size:%s) (modified:%s) exists",
		name, basePath, humanize.Bytes(uint64(info.Size())), info.ModTime())

	return DiskFile{
		name:     name,
		path:     filepath.Join(basePath, name),
		filesize: info.Size(),
		modified: info.ModTime(),
		isdir:    info.IsDir(),
		mode:     info.Mode(),
	}, nil
}

func FindDisk(dir string, recursive bool, fltr *filter.Filter) Files {
	var files Files
	dir = filepath.Clean(dir)
	if dir == "." || dir == "" {
		if pwd, err := os.Getwd(); err != nil {
			dir = filepath.Clean(dir)
		} else {
			dir = pwd
		}
	}

	var recursively string
	if recursive {
		recursively = " recursively"
	}

	log.Debugf("gonna find files%s in %s matching %s", recursively, dir, fltr)

	if recursive {
		files = append(files, walkDir(dir, fltr)...)
	} else {
		files = append(files, readDir(dir, fltr)...)
	}

	return files
}

// isInHiddenDir checks `path` and parent directories
// of `path` up to `base` for a hidden parent.
func isInHiddenDir(base, path string) bool {
	current := path
	for {
		current = filepath.Clean(current)
		if current == base {
			break
		}
		if strings.HasPrefix(filepath.Base(current), ".") {
			return true
		}
		current += string(os.PathSeparator) + ".."
	}
	return false
}

func walkDir(dir string, fltr *filter.Filter) Files {
	var files Files
	err := filepath.WalkDir(dir, func(path string, dirEntry fs.DirEntry, err error) error {
		if dir == path {
			return nil
		}

		if isInHiddenDir(dir, path) && fltr.IgnoreHidden() {
			return nil
		}

		actualPath, e := filepath.Abs(path)
		if e != nil {
			return err
		}

		name := dirEntry.Name()
		info, _ := dirEntry.Info()
		if fltr.Match(info) {
			files = append(files, DiskFile{
				path:     actualPath,
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
		return nil
	}
	return files
}

func readDir(dir string, fltr *filter.Filter) Files {
	var files Files
	fs, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, file := range fs {
		name := file.Name()

		if name == dir {
			continue
		}

		info, err := file.Info()
		if err != nil {
			return nil
		}

		path := filepath.Dir(filepath.Join(dir, name))

		if fltr.Match(info) {
			files = append(files, DiskFile{
				name:     name,
				path:     filepath.Join(path, name),
				modified: info.ModTime(),
				filesize: info.Size(),
				isdir:    info.IsDir(),
				mode:     info.Mode(),
			})
		}
	}
	return files
}
