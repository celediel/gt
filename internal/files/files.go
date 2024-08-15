// Package files finds and displays files on disk
package files

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type File interface {
	Name() string
	Path() string
	Date() time.Time
	Filesize() int64
	IsDir() bool
	Mode() fs.FileMode
	String() string
}

type Files []File

func (fls Files) String() string {
	var out = strings.Builder{}
	for _, file := range fls {
		out.WriteString(fmt.Sprintf("%s\t%s\t%s\n",
			file.Date().Format(time.RFC3339), file.Name(), file.Path(),
		))
	}
	return out.String()
}

func (fls Files) TotalSize() int64 {
	var size int64

	for _, file := range fls {
		if file.IsDir() {
			if d, ok := loadedDirSizes[file.Name()]; ok {
				log.Debugf("%s: got %d from directorysizes", file.Name(), d.size)
				size += d.size
				continue
			}
		}
		size += file.Filesize()
	}

	return size
}

func SortByModified(a, b File) int {
	if a.Date().Before(b.Date()) {
		return 1
	} else if a.Date().After(b.Date()) {
		return -1
	}
	return 0
}

func SortByModifiedReverse(a, b File) int {
	if a.Date().After(b.Date()) {
		return 1
	} else if a.Date().Before(b.Date()) {
		return -1
	}
	return 0
}

func SortBySize(a, b File) int {
	return cmp.Compare(a.Filesize(), b.Filesize())
}

func SortBySizeReverse(a, b File) int {
	return cmp.Compare(b.Filesize(), a.Filesize())
}

func SortByName(a, b File) int {
	return doNameSort(a, b)
}

func SortByNameReverse(a, b File) int {
	return doNameSort(b, a)
}

func SortByPath(a, b File) int {
	return cmp.Compare(a.Path(), b.Path())
}

func SortByPathReverse(a, b File) int {
	return cmp.Compare(b.Path(), a.Path())
}

func SortByExtension(a, b File) int {
	aext := strings.ToLower(filepath.Ext(a.Name()))
	bext := strings.ToLower(filepath.Ext(b.Name()))
	return cmp.Compare(aext, bext)
}

func SortByExtensionReverse(a, b File) int {
	aext := strings.ToLower(filepath.Ext(a.Name()))
	bext := strings.ToLower(filepath.Ext(b.Name()))
	return cmp.Compare(bext, aext)
}

func SortDirectoriesFirst(a, b File) int {
	if !a.IsDir() && b.IsDir() {
		return 1
	} else if a.IsDir() && !b.IsDir() {
		return -1
	}
	return 0
}

func SortDirectoriesLast(a, b File) int {
	if a.IsDir() && !b.IsDir() {
		return 1
	} else if !a.IsDir() && b.IsDir() {
		return -1
	}
	return 0
}

func doNameSort(a, b File) int {
	aname := strings.ToLower(a.Name())
	bname := strings.ToLower(b.Name())
	// check if filename is a number
	abase := strings.Replace(aname, filepath.Ext(aname), "", 1)
	bbase := strings.Replace(bname, filepath.Ext(bname), "", 1)
	ai, aerr := strconv.Atoi(abase)
	bi, berr := strconv.Atoi(bbase)
	if aerr == nil && berr == nil {
		return cmp.Compare(ai, bi)
	}
	return cmp.Compare(aname, bname)
}

func calculateDirSize(path string) int64 {
	var size int64
	info, err := os.Lstat(path)
	if err != nil {
		log.Error(err)
		return 0
	}
	if !info.IsDir() {
		return 0
	}

	files, err := os.ReadDir(path)
	if err != nil {
		log.Error(err)
		return 0
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		info, err := os.Lstat(filePath)
		if err != nil {
			log.Error(err)
			return 0
		}
		if info.IsDir() {
			size += calculateDirSize(filePath)
		} else {
			size += info.Size()
		}
	}

	return size
}
