// Package filter filters files based on specific critera
package filter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"github.com/ijt/go-anytime"
)

type Filter struct {
	on, before, after   time.Time
	glob, pattern       string
	unglob, unpattern   string
	filenames           []string
	dirsonly, filesonly bool
	showhidden          bool
	matcher             *regexp.Regexp
	unmatcher           *regexp.Regexp
	minsize, maxsize    int64
}

func (f *Filter) On() time.Time       { return f.on }
func (f *Filter) After() time.Time    { return f.after }
func (f *Filter) Before() time.Time   { return f.before }
func (f *Filter) Glob() string        { return f.glob }
func (f *Filter) Pattern() string     { return f.pattern }
func (f *Filter) FileNames() []string { return f.filenames }
func (f *Filter) FilesOnly() bool     { return f.filesonly }
func (f *Filter) DirsOnly() bool      { return f.dirsonly }
func (f *Filter) ShowHidden() bool    { return f.showhidden }
func (f *Filter) MinSize() int64      { return f.minsize }
func (f *Filter) MaxSize() int64      { return f.maxsize }

func (f *Filter) AddFileName(filename string) {
	filename = filepath.Clean(filename)
	f.filenames = append(f.filenames, filename)
}

func (f *Filter) AddFileNames(filenames ...string) {
	for _, filename := range filenames {
		f.AddFileName(filename)
	}
}

func (f *Filter) Match(filename string, modified time.Time, size int64, isdir bool) bool {
	// this might be unnessary but w/e
	filename = filepath.Clean(filename)

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

	if f.has_unregex() && f.unmatcher.MatchString(filename) {
		return false
	}

	if f.unglob != "" {
		if match, err := filepath.Match(f.unglob, filename); err != nil || match {
			return false
		}
	}

	if len(f.filenames) > 0 && !slices.Contains(f.filenames, filename) {
		return false
	}

	if f.filesonly && isdir {
		return false
	}

	if f.dirsonly && !isdir {
		return false
	}

	if !f.showhidden && strings.HasPrefix(filename, ".") {
		return false
	}

	if f.maxsize != 0 && f.maxsize < size {
		return false
	}

	if f.minsize != 0 && f.minsize > size {
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

func (f *Filter) SetUnPattern(unpattern string) error {
	var err error
	f.unpattern = unpattern
	f.unmatcher, err = regexp.Compile(f.unpattern)
	return err
}

func (f *Filter) Blank() bool {
	t := time.Time{}
	return !f.has_regex() &&
		!f.has_unregex() &&
		f.glob == "" &&
		f.unglob == "" &&
		f.after.Equal(t) &&
		f.before.Equal(t) &&
		f.on.Equal(t) &&
		len(f.filenames) == 0 &&
		!f.showhidden &&
		!f.filesonly &&
		!f.dirsonly &&
		f.minsize == 0 &&
		f.maxsize == 0
}

func (f *Filter) String() string {
	var m, unm string
	if f.matcher != nil {
		m = f.matcher.String()
	}
	if f.unmatcher != nil {
		unm = f.unmatcher.String()
	}
	return fmt.Sprintf("on:'%s' before:'%s' after:'%s' glob:'%s' regex:'%s' unglob:'%s' "+
		"unregex:'%s' filenames:'%v' filesonly:'%t' dirsonly:'%t' ignorehidden:'%t'",
		f.on, f.before, f.after,
		f.glob, m,
		f.unglob, unm,
		f.filenames,
		f.filesonly, f.dirsonly,
		f.showhidden,
	)
}

func (f *Filter) has_regex() bool {
	if f.matcher == nil {
		return false
	}
	return f.matcher.String() != ""
}

func (f *Filter) has_unregex() bool {
	if f.unmatcher == nil {
		return false
	}
	return f.unmatcher.String() != ""
}

func New(on, before, after, glob, pattern, unglob, unpattern string, filesonly, dirsonly, showhidden bool, minsize, maxsize string, names ...string) (*Filter, error) {
	var (
		err error
		now = time.Now()
	)

	f := &Filter{
		glob:       glob,
		unglob:     unglob,
		filesonly:  filesonly,
		dirsonly:   dirsonly,
		showhidden: showhidden,
	}

	f.AddFileNames(names...)

	if on != "" {
		o, err := anytime.Parse(on, now)
		if err != nil {
			return &Filter{}, err
		}
		f.on = o
	}

	if after != "" {
		a, err := anytime.Parse(after, now)
		if err != nil {
			return &Filter{}, err
		}
		f.after = a
	}

	if before != "" {
		b, err := anytime.Parse(before, now)
		if err != nil {
			return &Filter{}, err
		}
		f.before = b
	}

	err = f.SetPattern(pattern)
	if err != nil {
		return nil, err
	}
	err = f.SetUnPattern(unpattern)
	if err != nil {
		return nil, err
	}

	if minsize != "" {
		m, e := humanize.ParseBytes(minsize)
		if e != nil {
			log.Errorf("invalid input size '%s'", minsize)
		}
		f.minsize = int64(m)
	}

	if maxsize != "" {
		m, e := humanize.ParseBytes(maxsize)
		if e != nil {
			log.Errorf("invalid input size '%s'", maxsize)
		}
		f.maxsize = int64(m)
	}

	return f, nil
}

func same_day(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
