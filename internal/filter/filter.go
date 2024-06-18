// Package filter filters files based on specific critera
package filter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/ijt/go-anytime"
)

type Filter struct {
	on, before, after time.Time
	glob, pattern     string
	filenames         []string
	matcher           *regexp.Regexp
}

func (f *Filter) On() time.Time       { return f.on }
func (f *Filter) After() time.Time    { return f.after }
func (f *Filter) Before() time.Time   { return f.before }
func (f *Filter) Glob() string        { return f.glob }
func (f *Filter) Pattern() string     { return f.pattern }
func (f *Filter) FileNames() []string { return f.filenames }

func (f *Filter) Match(filename string, modified time.Time) bool {
	// on or before/after, not both
	if !f.on.IsZero() {
		if !same_day(f.on, modified) {
			return false
		}
	} else {
		if !f.after.IsZero() && f.after.After(modified) {
			return false
		}
		if !f.before.IsZero() && f.before.Before(modified) {
			return false
		}
	}
	if f.has_regex() && !f.matcher.MatchString(filename) {
		return false
	}
	if f.glob != "" {
		if match, err := filepath.Match(f.glob, filename); err != nil || !match {
			return false
		}
	}
	if len(f.filenames) > 0 && !slices.Contains(f.filenames, filename) {
		return false
	}
	// okay it was good
	return true
}

func (f *Filter) SetPattern(pattern string) error {
	var err error
	f.pattern = pattern
	f.matcher, err = regexp.Compile(f.pattern)
	return err
}

func (f *Filter) Blank() bool {
	t := time.Time{}
	return !f.has_regex() &&
		f.glob == "" &&
		f.after.Equal(t) &&
		f.before.Equal(t) &&
		f.on.Equal(t) &&
		len(f.filenames) == 0
}

func (f *Filter) String() string {
	var m string
	if f.matcher != nil {
		m = f.matcher.String()
	}
	return fmt.Sprintf("on:'%s' before:'%s' after:'%s' glob:'%s' regex:'%s' filenames:'%v'",
		f.on, f.before, f.after,
		f.glob, m, f.filenames,
	)
}

func (f *Filter) has_regex() bool {
	if f.matcher == nil {
		return false
	}
	return f.matcher.String() != ""
}

func New(o, b, a, g, p string, names ...string) (*Filter, error) {
	// o b a g p
	var (
		err error
		now = time.Now()
	)

	f := &Filter{
		glob:      g,
		filenames: append([]string{}, names...),
	}

	if o != "" {
		on, err := anytime.Parse(o, now)
		if err != nil {
			return &Filter{}, err
		}
		f.on = on
	}

	if a != "" {
		after, err := anytime.Parse(a, now)
		if err != nil {
			return &Filter{}, err
		}
		f.after = after
	}

	if b != "" {
		before, err := anytime.Parse(b, now)
		if err != nil {
			return &Filter{}, err
		}
		f.before = before
	}

	err = f.SetPattern(p)

	return f, err
}

func same_day(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
