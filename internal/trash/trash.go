// Package trash finds and displays files located in the trash, and moves
// files into the trash, creating cooresponding .trashinfo files
package trash

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.burning.moe/celediel/gt/internal/filter"
	"github.com/charmbracelet/log"
	"gitlab.com/tymonx/go-formatter/formatter"
	"gopkg.in/ini.v1"
)

const (
	trash_info_ext      string = ".trashinfo"
	trash_info_sec      string = "Trash Info"
	trash_info_path     string = "Path"
	trash_info_date     string = "DeletionDate"
	trash_info_date_fmt string = "2006-01-02T15:04:05"
	trash_info_template string = `[Trash Info]
Path={path}
DeletionDate={date}`
)

type Info struct {
	name, ogpath    string
	path, trashinfo string
	isdir           bool
	trashed         time.Time
	filesize        int64
}

type Infos []Info

func (i Info) Name() string       { return i.name }
func (i Info) Path() string       { return i.path }
func (i Info) OGPath() string     { return i.ogpath }
func (i Info) TrashInfo() string  { return i.trashinfo }
func (i Info) Trashed() time.Time { return i.trashed }
func (i Info) Filesize() int64    { return i.filesize }
func (i Info) IsDir() bool        { return i.isdir }

func FindFiles(trashdir, ogdir string, f *filter.Filter) (files Infos, outerr error) {
	outerr = filepath.WalkDir(trashdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Debugf("what happened?? what is %s?", err)
			return err
		}

		// ignore self, directories, and non trashinfo files
		if path == trashdir || d.IsDir() || filepath.Ext(path) != trash_info_ext {
			return nil
		}

		// trashinfo is just an ini file, so
		c, err := ini.Load(path)
		if err != nil {
			return err
		}
		if s := c.Section(trash_info_sec); s != nil {
			basepath := s.Key(trash_info_path).String()

			filename := filepath.Base(basepath)
			// maybe this is kind of a HACK
			trashedpath := strings.Replace(strings.Replace(path, "info", "files", 1), trash_info_ext, "", 1)
			info, _ := os.Stat(trashedpath)

			s := s.Key(trash_info_date).Value()
			date, err := time.ParseInLocation(trash_info_date_fmt, s, time.Local)
			if err != nil {
				return err
			}

			if ogdir != "" && filepath.Dir(basepath) != ogdir {
				return nil
			}

			if f.Match(filename, date, info.IsDir()) {
				log.Debugf("%s: deleted on %s", filename, date.Format(trash_info_date_fmt))
				files = append(files, Info{
					name:      filename,
					path:      trashedpath,
					ogpath:    basepath,
					trashinfo: path,
					trashed:   date,
					isdir:     info.IsDir(),
					filesize:  info.Size(),
				})
			} else {
				log.Debugf("(ignored) %s: deleted on %s", filename, date.Format(trash_info_date_fmt))
			}

		}
		return nil
	})
	if outerr != nil {
		return []Info{}, outerr
	}
	return
}

func Restore(files []Info) (restored int, err error) {
	for _, file := range files {
		log.Infof("restoring %s back to %s\n", file.name, file.ogpath)
		if err = os.Rename(file.path, file.ogpath); err != nil {
			return restored, err
		}
		if err = os.Remove(file.trashinfo); err != nil {
			return restored, err
		}
		restored++
	}
	return restored, err
}

func Remove(files []Info) (removed int, err error) {
	for _, file := range files {
		log.Infof("removing %s permanently forever!!!", file.name)
		if err = os.Remove(file.path); err != nil {
			if i, e := os.Stat(file.path); e == nil && i.IsDir() {
				err = os.RemoveAll(file.path)
				if err != nil {
					return removed, err
				}
			} else {
				return removed, err
			}
		}
		if err = os.Remove(file.trashinfo); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, err
}

func TrashFile(trashDir, name string) error {
	var (
		outdir   = filepath.Join(trashDir, "files")
		trashout = filepath.Join(trashDir, "info")
		filename = filepath.Base(name)
	)

	trashinfo_filename := filepath.Join(trashout, filename+trash_info_ext)
	out_path := filepath.Join(outdir, filename)

	if err := os.Rename(name, out_path); err != nil {
		return err
	}

	trash_info, err := formatter.Format(trash_info_template, formatter.Named{
		"path": name,
		"date": time.Now().Format(trash_info_date_fmt),
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(trashinfo_filename, []byte(trash_info), fs.FileMode(0600)); err != nil {
		return err
	}
	return nil
}

func TrashFiles(trashDir string, files ...string) (trashed int, err error) {
	for _, file := range files {
		if err = TrashFile(trashDir, file); err != nil {
			return trashed, err
		}
		trashed++
	}
	return trashed, err
}

func SortByTrashed(a, b Info) int {
	if a.trashed.After(b.trashed) {
		return 1
	} else if a.trashed.Before(b.trashed) {
		return -1
	} else {
		return 0
	}
}

func SortByTrashedReverse(a, b Info) int {
	if a.trashed.Before(b.trashed) {
		return 1
	} else if a.trashed.After(b.trashed) {
		return -1
	} else {
		return 0
	}
}

func SortBySize(a, b Info) int {
	if a.filesize > b.filesize {
		return 1
	} else if a.filesize < b.filesize {
		return -1
	} else {
		return 0
	}
}

func SortBySizeReverse(a, b Info) int {
	if a.filesize < b.filesize {
		return 1
	} else if a.filesize > b.filesize {
		return -1
	} else {
		return 0
	}
}
