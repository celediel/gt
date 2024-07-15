package files

import (
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.burning.moe/celediel/gt/internal/dirs"
	"git.burning.moe/celediel/gt/internal/filter"
	"git.burning.moe/celediel/gt/internal/prompt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"gitlab.com/tymonx/go-formatter/formatter"
	"gopkg.in/ini.v1"
)

const (
	random_str_length   int    = 8
	trash_info_ext      string = ".trashinfo"
	trash_info_sec      string = "Trash Info"
	trash_info_path     string = "Path"
	trash_info_date     string = "DeletionDate"
	trash_info_date_fmt string = "2006-01-02T15:04:05"
	trash_info_template string = `[Trash Info]
Path={path}
DeletionDate={date}
`
)

type TrashInfo struct {
	name, ogpath    string
	path, trashinfo string
	isdir           bool
	trashed         time.Time
	filesize        int64
}

func (t TrashInfo) Name() string      { return t.name }
func (t TrashInfo) TrashPath() string { return t.path }
func (t TrashInfo) Path() string      { return t.ogpath }
func (t TrashInfo) TrashInfo() string { return t.trashinfo }
func (t TrashInfo) Date() time.Time   { return t.trashed }
func (t TrashInfo) IsDir() bool       { return t.isdir }
func (t TrashInfo) Filesize() int64 {
	if t.isdir {
		return -1
	}
	return t.filesize
}

func (t TrashInfo) String() string {
	return fmt.Sprintf(string_format, t.name, t.path, t.trashed.Format(time.UnixDate), t.filesize, t.isdir)
}

func FindTrash(trashdir, ogdir string, f *filter.Filter) (files Files, outerr error) {
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
			trashedpath := strings.Replace(strings.Replace(path, "info", "files", 1), trash_info_ext, "", 1)
			info, err := os.Stat(trashedpath)
			if err != nil {
				log.Errorf("error reading %s: %s", trashedpath, err)
			}

			s := s.Key(trash_info_date).Value()
			date, err := time.ParseInLocation(trash_info_date_fmt, s, time.Local)
			if err != nil {
				return err
			}

			if ogdir != "" && filepath.Dir(basepath) != ogdir {
				return nil
			}

			if f.Match(filename, date, info.Size(), info.IsDir()) {
				log.Debugf("%s: deleted on %s", filename, date.Format(trash_info_date_fmt))
				files = append(files, TrashInfo{
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
		return Files{}, outerr
	}
	return
}

func Restore(files Files) (restored int, err error) {
	for _, maybeFile := range files {
		file, ok := maybeFile.(TrashInfo)
		if !ok {
			return restored, fmt.Errorf("bad file?? %s", maybeFile.Name())
		}

		var outpath string = dirs.UnEscape(file.ogpath)
		var cancel bool
		log.Infof("restoring %s back to %s\n", file.name, outpath)
		if _, e := os.Stat(outpath); e == nil {
			outpath, cancel = promptNewPath(outpath)
		}
		if cancel {
			continue
		}
		if err = os.Rename(file.path, outpath); err != nil {
			return restored, err
		}
		if err = os.Remove(file.trashinfo); err != nil {
			return restored, err
		}
		restored++
	}
	return restored, err
}

func Remove(files Files) (removed int, err error) {
	for _, maybeFile := range files {
		file, ok := maybeFile.(TrashInfo)
		if !ok {
			return removed, fmt.Errorf("bad file?? %s", maybeFile.Name())
		}

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
	trashinfo_filename, out_path := ensureUniqueName(filepath.Base(name), trashDir)

	// TODO: write across filesystems
	if err := os.Rename(name, out_path); err != nil {
		if strings.Contains(err.Error(), "invalid cross-device link") {
			return fmt.Errorf("not trashing file '%s': On different filesystem from trash directory", name)
		}
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

func randomFilename(length int) string {
	out := strings.Builder{}
	for range length {
		out.WriteByte(randomChar())
	}
	return out.String()
}

func randomChar() byte {
	const chars string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	return chars[rand.Intn(len(chars))]
}

func ensureUniqueName(filename, trashDir string) (string, string) {
	var (
		filedir = filepath.Join(trashDir, "files")
		infodir = filepath.Join(trashDir, "info")
	)

	info := filepath.Join(infodir, filename+trash_info_ext)
	if _, err := os.Stat(info); os.IsNotExist(err) {
		// doesn't exist, so use it
		path := filepath.Join(filedir, filename)
		return info, path
	} else {
		// otherwise, try random suffixes until one works
		log.Debugf("%s exists in trash, generating random name", filename)
		var tries int
		for {
			tries++
			rando := randomFilename(random_str_length)
			new_name := filepath.Join(infodir, filename+rando+trash_info_ext)
			if _, err := os.Stat(new_name); os.IsNotExist(err) {
				path := filepath.Join(filedir, filename+rando)
				log.Debugf("settled on random name %s%s on the %s try", filename, rando, humanize.Ordinal(tries))
				return new_name, path
			}
		}
	}
}

func promptNewPath(path string) (string, bool) {
	for {
		answer := prompt.AskRune(fmt.Sprintf("file %s exists, overwrite, rename, or cancel?", path), "o/r/c")
		switch answer {
		case 'o', 'O':
			return path, false
		case 'r', 'R':
			if err := huh.NewInput().
				Title("input a new filename").
				Value(&path).
				Run(); err != nil {
				return path, false
			}
			if _, e := os.Stat(path); e != nil {
				return path, false
			}
		default:
			return path, true
		}
	}
}
