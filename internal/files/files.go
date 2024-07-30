// Package files finds and displays files on disk
package files

import (
	"cmp"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

func SortByModified(a, b File) int {
	if a.Date().Before(b.Date()) {
		return 1
	} else if a.Date().After(b.Date()) {
		return -1
	} else {
		return 0
	}
}

func SortByModifiedReverse(a, b File) int {
	if a.Date().After(b.Date()) {
		return 1
	} else if a.Date().Before(b.Date()) {
		return -1
	} else {
		return 0
	}
}

func SortBySize(a, b File) int {
	as := getSortingSize(a)
	bs := getSortingSize(b)
	return cmp.Compare(as, bs)
}

func SortBySizeReverse(a, b File) int {
	as := getSortingSize(a)
	bs := getSortingSize(b)
	return cmp.Compare(bs, as)
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
	} else {
		return 0
	}
}

func SortDirectoriesLast(a, b File) int {
	if a.IsDir() && !b.IsDir() {
		return 1
	} else if !a.IsDir() && b.IsDir() {
		return -1
	} else {
		return 0
	}
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

func getSortingSize(f File) int64 {
	if f.IsDir() {
		return -1
	} else {
		return f.Filesize()
	}
}
