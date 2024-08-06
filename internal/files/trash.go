package files

import (
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"git.burning.moe/celediel/gt/internal/dirs"
	"git.burning.moe/celediel/gt/internal/filter"
	"git.burning.moe/celediel/gt/internal/prompt"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"github.com/moby/sys/mountinfo"
	"gitlab.com/tymonx/go-formatter/formatter"
	"gopkg.in/ini.v1"
)

const (
	sep                      = string(os.PathSeparator)
	executePerm              = fs.FileMode(0755)
	noExecuteUserPerm        = fs.FileMode(0600)
	randomStrLength   int    = 8
	trashName         string = ".Trash"
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

func FindInAllTrashes(ogdir string, fltr *filter.Filter) (Files, error) {
	var files Files

	personalTrash := filepath.Join(xdg.DataHome, "Trash")

	if fls, err := findTrash(personalTrash, ogdir, fltr); err == nil {
		files = append(files, fls...)
	}

	for _, trash := range getAllTrashes() {
		fls, err := findTrash(trash, ogdir, fltr)
		if err != nil {
			continue
		}
		files = append(files, fls...)
	}

	return files, nil
}

func ConfirmRestore(confirm bool, fs Files) error {
	if !confirm || prompt.YesNo(fmt.Sprintf("restore %d selected files?", len(fs))) {
		log.Info("doing the thing")
		restored, err := restore(fs)
		if err != nil {
			return fmt.Errorf("restored %d files before error %w", restored, err)
		}
		fmt.Fprintf(os.Stdout, "restored %d files\n", restored)
	} else {
		fmt.Fprintf(os.Stdout, "not doing anything\n")
	}
	return nil
}

func ConfirmClean(confirm bool, fs Files) error {
	if prompt.YesNo(fmt.Sprintf("remove %d selected files permanently from the trash?", len(fs))) &&
		(!confirm || prompt.YesNo(fmt.Sprintf("really remove all these %d selected files permanently from the trash forever??", len(fs)))) {
		removed, err := remove(fs)
		if err != nil {
			return fmt.Errorf("removed %d files before error %w", removed, err)
		}
		fmt.Fprintf(os.Stdout, "removed %d files\n", removed)
	} else {
		fmt.Fprintf(os.Stdout, "not doing anything\n")
	}
	return nil
}

func ConfirmTrash(confirm bool, fs Files) error {
	if !confirm || prompt.YesNo(fmt.Sprintf("trash %d selected files?", len(fs))) {
		tfs := make([]string, 0, len(fs))
		for _, file := range fs {
			tfs = append(tfs, file.Path())
		}

		trashed := trashFiles(tfs)

		var s string
		if trashed > 1 {
			s = "s"
		}
		fmt.Fprintf(os.Stdout, "trashed %d file%s\n", trashed, s)
	} else {
		fmt.Fprintf(os.Stdout, "not doing anything\n")
		return nil
	}
	return nil
}

func findTrash(trashdir, ogdir string, fltr *filter.Filter) (Files, error) {
	log.Debugf("searching for trashinfo files in %s", trashdir)
	var files Files

	infodir := filepath.Join(trashdir, "info")
	dirs, err := os.ReadDir(infodir)
	if err != nil {
		return Files{}, err
	}

	for _, dir := range dirs {
		if dir.IsDir() || filepath.Ext(dir.Name()) != trashInfoExt {
			continue
		}

		path := filepath.Join(infodir, dir.Name())

		// trashinfo is just an ini file, so
		trashInfo, err := ini.Load(path)
		if err != nil {
			log.Errorf("error reading %s: %s", path, err)
			continue
		}

		if section := trashInfo.Section(trashInfoSec); section != nil {
			basepath := section.Key(trashInfoPath).String()

			filename := filepath.Base(basepath)
			trashedpath := strings.Replace(strings.Replace(path, "info", "files", 1), trashInfoExt, "", 1)
			info, err := os.Lstat(trashedpath)
			if err != nil {
				log.Errorf("error reading '%s': %s", trashedpath, err)
				continue
			}

			s := section.Key(trashInfoDate).Value()
			date, err := time.ParseInLocation(trashInfoDateFmt, s, time.Local)
			if err != nil {
				log.Errorf("error parsing date '%s' in trashinfo file '%s': %s", s, path, err)
				continue
			}

			if ogdir != "" && filepath.Dir(basepath) != ogdir {
				continue
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
	}

	return files, nil
}

func trashFile(filename string) error {
	trashDir, err := getTrashDir(filename)
	if err != nil {
		return err
	}

	trashInfoFilename, outPath := getTrashFilenames(filepath.Base(filename), trashDir)

	if err := os.Rename(filename, outPath); err != nil {
		return err
	}

	trashInfo, err := formatter.Format(trashInfoTemplate, formatter.Named{
		"path": filename,
		"date": time.Now().Format(trashInfoDateFmt),
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(trashInfoFilename, []byte(trashInfo), noExecuteUserPerm); err != nil {
		return err
	}

	return nil
}

func trashFiles(files []string) (trashed int) {
	for _, file := range files {
		if err := trashFile(file); err != nil {
			log.Errorf("cannot trash '%s': %s", file, err)
			continue
		}
		trashed++
	}
	return
}

func restore(files Files) (restored int, err error) {
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

func remove(files Files) (removed int, err error) {
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

func randomString(length int) string {
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

func getTrashFilenames(filename, trashDir string) (string, string) {
	var (
		filedir = filepath.Join(trashDir, "files")
		infodir = filepath.Join(trashDir, "info")
	)

	info := filepath.Join(infodir, filename+trashInfoExt)
	if _, err := os.Stat(info); os.IsNotExist(err) {
		// doesn't exist, so use it
		path := filepath.Join(filedir, filename)
		return info, path
	}

	// otherwise, try random suffixes until one works
	log.Debugf("%s exists in trash, generating random name", filename)
	var tries int
	for {
		tries++
		rando := randomString(randomStrLength)
		newInfo := filepath.Join(infodir, filename+rando+trashInfoExt)
		newFile := filepath.Join(filedir, filename+rando)
		_, infoErr := os.Stat(newInfo)
		_, fileErr := os.Stat(newFile)
		if os.IsNotExist(infoErr) && os.IsNotExist(fileErr) {
			path := filepath.Join(filedir, filename+rando)
			log.Debugf("settled on random name %s%s on the %s try", filename, rando, humanize.Ordinal(tries))
			return newInfo, path
		}
	}
}

func getTrashDir(filename string) (string, error) {
	root, err := getRoot(filename)
	if err != nil {
		return "", err
	}

	var trashDir string
	if strings.Contains(filename, xdg.Home) {
		trashDir = filepath.Join(xdg.DataHome, trashName[1:])
	} else {
		trashDir = filepath.Clean(root + sep + trashName)
	}

	if _, err := os.Lstat(trashDir); err != nil {
		usr, _ := user.Current()
		trashDir += "-" + usr.Uid
		if err := os.Mkdir(trashDir, executePerm); err != nil {
			return "", fmt.Errorf("%s%s does not exist and creation of %s failed", root, trashName, trashDir)
		}
	}

	if link, err := os.Readlink(trashDir); err == nil && link != "" {
		return "", fmt.Errorf("trash dir %s is a symbolic link", trashDir)
	}

	return trashDir, nil
}

func getRoot(path string) (string, error) {
	var roots []string

	// populate a list of mountpoints on the system
	_, err := mountinfo.GetMounts(func(i *mountinfo.Info) (skip bool, stop bool) {
		roots = append(roots, i.Mountpoint)
		return false, false
	})
	if err != nil {
		log.Error(err)
	}

	var depth uint8 = 1 // 255 seems a reasonable recursion maximum
	current := path

	// recursively search upwards by using filepath.Clean and ..
	for {
		if depth == 0 {
			return path, fmt.Errorf("reached max depth getting root of %s", path)
		}

		current = filepath.Clean(current)

		if current == string(os.PathSeparator) || slices.Contains(roots, current) {
			return current, nil
		}

		current += string(os.PathSeparator) + ".."
		depth++
	}
}

func getAllTrashes() []string {
	var trashes []string
	usr, _ := user.Current()

	_, err := mountinfo.GetMounts(func(mount *mountinfo.Info) (skip bool, stop bool) {
		point := mount.Mountpoint
		trashDir := filepath.Clean(point + sep + trashName)
		userTrashDir := trashDir + "-" + usr.Uid

		if _, err := os.Lstat(trashDir); err == nil {
			trashes = append(trashes, trashDir)
		}

		if _, err := os.Lstat(userTrashDir); err == nil {
			trashes = append(trashes, userTrashDir)
		}

		return false, false
	})

	if err != nil {
		return []string{}
	}

	return trashes
}
