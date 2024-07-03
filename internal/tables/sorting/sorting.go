package sorting

import "git.burning.moe/celediel/gt/internal/files"

type Sorting int

const (
	Name Sorting = iota + 1
	NameReverse
	Date
	DateReverse
	Path
	PathReverse
	Size
	SizeReverse
)

func (s Sorting) Next() Sorting {
	switch s {
	case SizeReverse:
		return Name
	default:
		return s + 1
	}
}

func (s Sorting) Prev() Sorting {
	switch s {
	case Name:
		return SizeReverse
	default:
		return s - 1
	}
}

func (s Sorting) String() string {
	switch s {
	case Name:
		return "name"
	case NameReverse:
		return "name (r)"
	case Date:
		return "date"
	case DateReverse:
		return "date (r)"
	case Path:
		return "path"
	case PathReverse:
		return "path (r)"
	case Size:
		return "size"
	case SizeReverse:
		return "size (r)"
	default:
		return "0"
	}
}

func (s Sorting) Sorter() func(a, b files.File) int {
	switch s {
	case Name:
		return files.SortByName
	case NameReverse:
		return files.SortByNameReverse
	case Date:
		return files.SortByModified
	case DateReverse:
		return files.SortByModifiedReverse
	case Path:
		return files.SortByPath
	case PathReverse:
		return files.SortByPathReverse
	case Size:
		return files.SortBySize
	case SizeReverse:
		return files.SortBySizeReverse
	default:
		return files.SortByName
	}
}
