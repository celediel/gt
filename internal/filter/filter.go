// Package filter filters files based on specific criteria
package filter

import (
	"fmt"
	"io/fs"
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
	ignorehidden        bool
	matcher             *regexp.Regexp
	unmatcher           *regexp.Regexp
	minsize, maxsize    int64
	mode                fs.FileMode
}

func (f *Filter) On() time.Time       { return f.on }
func (f *Filter) After() time.Time    { return f.after }
func (f *Filter) Before() time.Time   { return f.before }
func (f *Filter) Glob() string        { return f.glob }
func (f *Filter) Pattern() string     { return f.pattern }
func (f *Filter) FileNames() []string { return f.filenames }
func (f *Filter) FilesOnly() bool     { return f.filesonly }
func (f *Filter) DirsOnly() bool      { return f.dirsonly }
func (f *Filter) IgnoreHidden() bool  { return f.ignorehidden }
func (f *Filter) MinSize() int64      { return f.minsize }
func (f *Filter) MaxSize() int64      { return f.maxsize }
func (f *Filter) Mode() fs.FileMode   { return f.mode }

func (f *Filter) AddFileName(filename string) {
	filename = filepath.Clean(filename)
	f.filenames = append(f.filenames, filename)
}

func (f *Filter) AddFileNames(filenames ...string) {
	for _, filename := range filenames {
		f.AddFileName(filename)
	}
}

func (f *Filter) Match(info fs.FileInfo) bool {
	filename := info.Name()
	modified := info.ModTime()
	isdir := info.IsDir()
	size := info.Size()
	mode := info.Mode()

	// on or before/after, not both
	if !f.on.IsZero() {
		if !sameDay(f.on, modified) {
			log.Debugf("%s: %s isn't on %s, bye!", filename, modified, f.on)
			return false
		}
	} else {
		if !f.after.IsZero() && f.after.After(modified) {
			log.Debugf("%s: %s isn't after %s, bye!", filename, modified, f.after)
			return false
		}
		if !f.before.IsZero() && f.before.Before(modified) {
			log.Debugf("%s: %s isn't before %s, bye!", filename, modified, f.before)
			return false
		}
	}

	if f.hasRegex() && !f.matcher.MatchString(filename) {
		log.Debugf("%s doesn't match `%s`, bye!", filename, f.matcher.String())
		return false
	}

	if f.glob != "" {
		if match, err := filepath.Match(f.glob, filename); err != nil || !match {
			log.Debugf("%s doesn't match `%s`, bye!", filename, f.glob)
			return false
		}
	}

	if f.hasUnregex() && f.unmatcher.MatchString(filename) {
		log.Debugf("%s matches `%s`, bye!", filename, f.unmatcher.String())
		return false
	}

	if f.unglob != "" {
		if match, err := filepath.Match(f.unglob, filename); err != nil || match {
			log.Debugf("%s matches `%s`, bye!", filename, f.unglob)
			return false
		}
	}

	if len(f.filenames) > 0 && !slices.Contains(f.filenames, filename) {
		log.Debugf("%s isn't in %v, bye!", filename, f.filenames)
		return false
	}

	if f.filesonly && isdir {
		log.Debugf("%s is dir, bye!", filename)
		return false
	}

	if f.dirsonly && !isdir {
		log.Debugf("%s is file, bye!", filename)
		return false
	}

	if f.ignorehidden && strings.HasPrefix(filename, ".") {
		log.Debugf("%s is hidden, bye!", filename)
		return false
	}

	if f.maxsize != 0 && f.maxsize < size {
		log.Debugf("%s is larger than %d, bye!", filename, f.maxsize)
		return false
	}

	if f.minsize != 0 && f.minsize > size {
		log.Debugf("%s is smaller than %d, bye!", filename, f.minsize)
		return false
	}

	if f.mode != 0 && f.mode != mode && f.mode-fs.ModeDir != mode {
		log.Debugf("%s mode:'%s' (%d) isn't '%s' (%d), bye!", filename, mode, mode, f.mode, f.mode)
		return false
	}

	// okay it was good
	log.Debugf("%s modified:'%s' dir:'%T' mode:'%s' was a good one!", filename, modified, isdir, mode)
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
	blank := time.Time{}
	return !f.hasRegex() &&
		!f.hasUnregex() &&
		f.glob == "" &&
		f.unglob == "" &&
		f.after.Equal(blank) &&
		f.before.Equal(blank) &&
		f.on.Equal(blank) &&
		len(f.filenames) == 0 &&
		!f.ignorehidden &&
		!f.filesonly &&
		!f.dirsonly &&
		f.minsize == 0 &&
		f.maxsize == 0 &&
		f.mode == 0
}

func (f *Filter) String() string {
	var match, unmatch string
	if f.matcher != nil {
		match = f.matcher.String()
	}
	if f.unmatcher != nil {
		unmatch = f.unmatcher.String()
	}
	return fmt.Sprintf("on:'%s' before:'%s' after:'%s' glob:'%s' regex:'%s' unglob:'%s' "+
		"unregex:'%s' filenames:'%v' filesonly:'%t' dirsonly:'%t' ignorehidden:'%t' "+
		"minsize:'%d' maxsize:'%d' mode:'%s'",
		f.on, f.before, f.after,
		f.glob, match, f.unglob, unmatch,
		f.filenames, f.filesonly, f.dirsonly,
		f.ignorehidden, f.minsize, f.maxsize, f.mode,
	)
}

func (f *Filter) hasRegex() bool {
	if f.matcher == nil {
		return false
	}
	return f.matcher.String() != ""
}

func (f *Filter) hasUnregex() bool {
	if f.unmatcher == nil {
		return false
	}
	return f.unmatcher.String() != ""
}

func New(on, before, after, glob, pattern, unglob, unpattern string, filesonly, dirsonly, ignorehidden bool, minsize, maxsize string, mode fs.FileMode, names ...string) (*Filter, error) {
	var (
		err error
		now = time.Now()
	)

	filter := &Filter{
		glob:         glob,
		unglob:       unglob,
		filesonly:    filesonly,
		dirsonly:     dirsonly,
		ignorehidden: ignorehidden,
		mode:         mode,
	}

	filter.AddFileNames(names...)

	if on != "" {
		o, err := anytime.Parse(on, now)
		if err != nil {
			return &Filter{}, err
		}
		filter.on = o
	}

	if after != "" {
		a, err := anytime.Parse(after, now)
		if err != nil {
			return &Filter{}, err
		}
		filter.after = a
	}

	if before != "" {
		b, err := anytime.Parse(before, now)
		if err != nil {
			return &Filter{}, err
		}
		filter.before = b
	}

	err = filter.SetPattern(pattern)
	if err != nil {
		return nil, err
	}
	err = filter.SetUnPattern(unpattern)
	if err != nil {
		return nil, err
	}

	if minsize != "" {
		m, e := humanize.ParseBytes(minsize)
		if e != nil {
			log.Errorf("invalid input size '%s'", minsize)
		}
		filter.minsize = int64(m)
	}

	if maxsize != "" {
		m, e := humanize.ParseBytes(maxsize)
		if e != nil {
			log.Errorf("invalid input size '%s'", maxsize)
		}
		filter.maxsize = int64(m)
	}

	return filter, nil
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
