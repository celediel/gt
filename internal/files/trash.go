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

	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"gitlab.com/tymonx/go-formatter/formatter"
	"gopkg.in/ini.v1"
)

const (
	executePerm              = fs.FileMode(0755)
	noExecuteUserPerm        = fs.FileMode(0600)
	randomStrLength   int    = 8
	trashInfoExt      string = ".trashinfo"
	trashInfoSec      string = "Trash Info"
	trashInfoPath     string = "Path"
	trashInfoDate     string = "DeletionDate"
	trashInfoDateFmt  string = "2006-01-02T15:04:05"
	trashInfoTemplate string = `[Trash Info]
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
	mode            fs.FileMode
}

func (t TrashInfo) Name() string      { return t.name }
func (t TrashInfo) TrashPath() string { return t.path }
func (t TrashInfo) Path() string      { return t.ogpath }
func (t TrashInfo) TrashInfo() string { return t.trashinfo }
func (t TrashInfo) Date() time.Time   { return t.trashed }
func (t TrashInfo) IsDir() bool       { return t.isdir }
func (t TrashInfo) Mode() fs.FileMode { return t.mode }
func (t TrashInfo) Filesize() int64 {
	if t.isdir {
		return 0
	}
	return t.filesize
}

func (t TrashInfo) String() string {
	return t.name + t.path + t.ogpath + t.trashinfo
}

func FindTrash(trashdir, ogdir string, fltr *filter.Filter) (Files, error) {
	var files Files
	outerr := filepath.WalkDir(trashdir, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			log.Debugf("what happened?? what is %s?", err)
			return err
		}

		// ignore self, directories, and non trashinfo files
		if path == trashdir || dirEntry.IsDir() || filepath.Ext(path) != trashInfoExt {
			return nil
		}

		// trashinfo is just an ini file, so
		c, err := ini.Load(path)
		if err != nil {
			return err
		}
		if section := c.Section(trashInfoSec); section != nil {
			basepath := section.Key(trashInfoPath).String()

			filename := filepath.Base(basepath)
			trashedpath := strings.Replace(strings.Replace(path, "info", "files", 1), trashInfoExt, "", 1)
			info, err := os.Lstat(trashedpath)
			if err != nil {
				log.Errorf("error reading %s: %s", trashedpath, err)
				return nil
			}

			s := section.Key(trashInfoDate).Value()
			date, err := time.ParseInLocation(trashInfoDateFmt, s, time.Local)
			if err != nil {
				return err
			}

			if ogdir != "" && filepath.Dir(basepath) != ogdir {
				return nil
			}

			if fltr.Match(info) {
				files = append(files, TrashInfo{
					name:      filename,
					path:      trashedpath,
					ogpath:    basepath,
					trashinfo: path,
					trashed:   date,
					isdir:     info.IsDir(),
					filesize:  info.Size(),
				})
			}
		}
		return nil
	})
	if outerr != nil {
		return Files{}, outerr
	}
	return files, nil
}

func Restore(files Files) (restored int, err error) {
	for _, maybeFile := range files {
		file, ok := maybeFile.(TrashInfo)
		if !ok {
			return restored, fmt.Errorf("bad file?? %s", maybeFile.Name())
		}

		var cancel bool
		outpath := dirs.UnEscape(file.ogpath)
		log.Infof("restoring %s back to %s\n", file.name, outpath)
		if _, e := os.Lstat(outpath); e == nil {
			outpath, cancel = prompt.NewPath(outpath)
		}

		if cancel {
			continue
		}

		basedir := filepath.Dir(outpath)
		if _, e := os.Stat(basedir); e != nil {
			if err = os.MkdirAll(basedir, executePerm); err != nil {
				return restored, err
			}
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

func ConfirmRestore(confirm bool, fs Files) error {
	if !confirm || prompt.YesNo(fmt.Sprintf("restore %d selected files?", len(fs))) {
		log.Info("doing the thing")
		restored, err := Restore(fs)
		if err != nil {
			return fmt.Errorf("restored %d files before error %w", restored, err)
		}
		fmt.Fprintf(os.Stdout, "restored %d files\n", restored)
	} else {
		fmt.Fprintf(os.Stdout, "not doing anything\n")
	}
	return nil
}

func Remove(files Files) (removed int, err error) {
	for _, maybeFile := range files {
		file, ok := maybeFile.(TrashInfo)
		if !ok {
			return removed, fmt.Errorf("bad file?? %s", maybeFile.Name())
		}

		if err = os.Remove(file.path); err != nil {
			if i, e := os.Lstat(file.path); e == nil && i.IsDir() {
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

func ConfirmClean(confirm bool, fs Files) error {
	if prompt.YesNo(fmt.Sprintf("remove %d selected files permanently from the trash?", len(fs))) &&
		(!confirm || prompt.YesNo(fmt.Sprintf("really remove all these %d selected files permanently from the trash forever??", len(fs)))) {
		removed, err := Remove(fs)
		if err != nil {
			return fmt.Errorf("removed %d files before error %w", removed, err)
		}
		fmt.Fprintf(os.Stdout, "removed %d files\n", removed)
	} else {
		fmt.Fprintf(os.Stdout, "not doing anything\n")
	}
	return nil
}

func TrashFile(trashDir, name string) error {
	trashinfoFilename, outPath := ensureUniqueName(filepath.Base(name), trashDir)

	// TODO: write across filesystems
	if err := os.Rename(name, outPath); err != nil {
		if strings.Contains(err.Error(), "invalid cross-device link") {
			return fmt.Errorf("not trashing file '%s': On different filesystem from trash directory", name)
		}
		return err
	}

	trashInfo, err := formatter.Format(trashInfoTemplate, formatter.Named{
		"path": name,
		"date": time.Now().Format(trashInfoDateFmt),
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(trashinfoFilename, []byte(trashInfo), noExecuteUserPerm); err != nil {
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

func ConfirmTrash(confirm bool, fs Files, trashDir string) error {
	if !confirm || prompt.YesNo(fmt.Sprintf("trash %d selected files?", len(fs))) {
		tfs := make([]string, 0, len(fs))
		for _, file := range fs {
			log.Debugf("gonna trash %s", file.Path())
			tfs = append(tfs, file.Path())
		}

		trashed, err := TrashFiles(trashDir, tfs...)
		if err != nil {
			return err
		}
		var files string
		if trashed == 1 {
			files = "file"
		} else {
			files = "files"
		}
		fmt.Fprintf(os.Stdout, "trashed %d %s\n", trashed, files)
	} else {
		fmt.Fprintf(os.Stdout, "not doing anything\n")
		return nil
	}
	return nil
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

	info := filepath.Join(infodir, filename+trashInfoExt)
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
			rando := randomFilename(randomStrLength)
			newName := filepath.Join(infodir, filename+rando+trashInfoExt)
			if _, err := os.Stat(newName); os.IsNotExist(err) {
				path := filepath.Join(filedir, filename+rando)
				log.Debugf("settled on random name %s%s on the %s try", filename, rando, humanize.Ordinal(tries))
				return newName, path
			}
		}
	}
}
