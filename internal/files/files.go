// Package files finds and displays files on disk
package files

import "time"

type File interface {
	Name() string
	Path() string
	Date() time.Time
	Filesize() int64
	IsDir() bool
}

type Files []File

func SortByModified(a, b File) int {
	if a.Date().After(b.Date()) {
		return 1
	} else if a.Date().Before(b.Date()) {
		return -1
	} else {
		return 0
	}
}

func SortByModifiedReverse(a, b File) int {
	if a.Date().Before(b.Date()) {
		return 1
	} else if a.Date().After(b.Date()) {
		return -1
	} else {
		return 0
	}
}

func SortBySize(a, b File) int {
	if a.Filesize() > b.Filesize() {
		return 1
	} else if a.Filesize() < b.Filesize() {
		return -1
	} else {
		return 0
	}
}

func SortBySizeReverse(a, b File) int {
	if a.Filesize() < b.Filesize() {
		return 1
	} else if a.Filesize() > b.Filesize() {
		return -1
	} else {
		return 0
	}
}

func SortByName(a, b File) int {
	if a.Name() > b.Name() {
		return 1
	} else if a.Name() < b.Name() {
		return -1
	} else {
		return 0
	}
}

func SortByNameReverse(a, b File) int {
	if a.Name() < b.Name() {
		return 1
	} else if a.Name() > b.Name() {
		return -1
	} else {
		return 0
	}
}

func SortByPath(a, b File) int {
	if a.Path() > b.Path() {
		return 1
	} else if a.Path() < b.Path() {
		return -1
	} else {
		return 0
	}
}

func SortByPathReverse(a, b File) int {
	if a.Path() < b.Path() {
		return 1
	} else if a.Path() > b.Path() {
		return -1
	} else {
		return 0
	}
}
