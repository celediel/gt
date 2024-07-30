// Package modes implements Mode type for interactive table.
package modes

type Mode int

const (
	Trashing Mode = iota + 1
	Listing
	Restoring
	Cleaning
	Interactive
)

func (m Mode) String() string {
	switch m {
	case Trashing:
		return "Trashing"
	case Listing:
		return "Listing"
	case Restoring:
		return "Restoring"
	case Cleaning:
		return "Cleaning"
	case Interactive:
		return "Interactive"
	default:
		return "0"
	}
}
