package filter_test

import (
	"fmt"
	"io/fs"
	"math"
	"testing"
	"time"

	"git.burning.moe/celediel/gt/internal/filter"
)

var (
	now           = time.Now()
	yesterday     = now.AddDate(0, 0, -1)
	ereyesterday  = now.AddDate(0, 0, -2)
	oneweekago    = now.AddDate(0, 0, -7)
	twoweeksago   = now.AddDate(0, 0, -14)
	onemonthago   = now.AddDate(0, -1, 0)
	twomonthsago  = now.AddDate(0, -2, 0)
	fourmonthsago = now.AddDate(0, -4, 0)
	oneyearago    = now.AddDate(-1, 0, 0)
	twoyearsago   = now.AddDate(-2, 0, 0)
	fouryearsago  = now.AddDate(-4, 0, 0)
)

type testholder struct {
	pattern, glob       string
	unpattern, unglob   string
	before, after, on   string
	filenames           []string
	filesonly, dirsonly bool
	ignorehidden        bool
	good, bad           []singletest
	minsize, maxsize    string
	mode                fs.FileMode
}

func (t testholder) String() string {
	return fmt.Sprintf(
		"pattern:'%s' glob:'%s' unpattern:'%s' unglob:'%s' filenames:'%v' "+
			"before:'%s' after:'%s' on:'%s' filesonly:'%t' dirsonly:'%t' "+
			"showhidden:'%t' minsize:'%s' maxsize:'%s'",
		t.pattern, t.glob, t.unpattern, t.unglob, t.filenames, t.before, t.after, t.on,
		t.filesonly, t.dirsonly, t.ignorehidden, t.minsize, t.maxsize,
	)
}

type singletest struct {
	filename string
	isdir    bool
	modified time.Time
	size     int64
	mode     fs.FileMode
}

func (s singletest) Name() string       { return s.filename }
func (s singletest) Size() int64        { return s.size }
func (s singletest) Mode() fs.FileMode  { return s.mode }
func (s singletest) ModTime() time.Time { return s.modified }
func (s singletest) IsDir() bool        { return s.isdir }
func (s singletest) Sys() any           { return nil }

func (s singletest) String() string {
	return fmt.Sprintf("filename:'%s' modified:'%s' size:'%d' isdir:'%t'", s.filename, s.modified, s.size, s.isdir)
}

func testmatch(t *testing.T, testers []testholder) {
	t.Helper() // I don't think this is a helper function but w/e
	const testnamefmt string = "file %s modified on %s"
	var (
		fltr *filter.Filter
		err  error
	)
	for _, tester := range testers {
		fltr, err = filter.New(
			tester.on, tester.before, tester.after, tester.glob, tester.pattern,
			tester.unglob, tester.unpattern, tester.filesonly, tester.dirsonly,
			tester.ignorehidden, tester.minsize, tester.maxsize, tester.mode,
			tester.filenames...,
		)
		if err != nil {
			t.Fatal(err)
		}

		for _, tst := range tester.good {
			t.Run(fmt.Sprintf(testnamefmt+"_good", tst.filename, tst.modified), func(t *testing.T) {
				if !fltr.Match(tst) {
					t.Fatalf("(%s) didn't match (%s) but should have", tst, tester)
				}
			})
		}

		for _, tst := range tester.bad {
			t.Run(fmt.Sprintf(testnamefmt+"_bad", tst.filename, tst.modified), func(t *testing.T) {
				if fltr.Match(tst) {
					t.Fatalf("(%s) matched (%s) but shouldn't have", tst, tester)
				}
			})
		}
	}
}

func nameonly(dir bool, names ...string) []singletest {
	out := make([]singletest, 0, len(names))
	for _, name := range names {
		out = append(out, singletest{filename: name, modified: time.Time{}, isdir: dir, size: 0, mode: 0000})
	}
	return out
}

func timeonly(dir bool, times ...time.Time) []singletest {
	out := make([]singletest, 0, len(times))
	for _, time := range times {
		out = append(out, singletest{filename: "blank.txt", modified: time, isdir: dir, size: 0, mode: 0000})
	}
	return out
}

func sizeonly(sizes ...int64) []singletest {
	out := make([]singletest, 0, len(sizes))
	for _, size := range sizes {
		out = append(out, singletest{filename: "blank", modified: time.Time{}, isdir: false, size: size, mode: 0000})
	}
	return out
}

func modeonly(dir bool, modes ...fs.FileMode) []singletest {
	out := make([]singletest, 0, len(modes))
	for _, mode := range modes {
		out = append(out, singletest{filename: "blank", modified: time.Time{}, isdir: dir, size: 0, mode: mode})
	}
	return out
}

func TestFilterOn(t *testing.T) {
	testmatch(t, []testholder{
		{
			on:   "2024-02-14",
			good: timeonly(false, time.Date(2024, 2, 14, 12, 0, 0, 0, time.Local)),
			bad:  timeonly(false, now, now.Add(time.Hour*72), now.Add(-time.Hour*18)),
		},
		{
			on:   "yesterday",
			good: timeonly(false, yesterday),
			bad:  timeonly(false, now, oneweekago, onemonthago, oneyearago, twoweeksago, twomonthsago, twoyearsago),
		},
		{
			on:   "one week ago",
			good: timeonly(false, oneweekago),
			bad:  timeonly(false, now),
		},
		{
			on:   "one month ago",
			good: timeonly(false, onemonthago),
			bad:  timeonly(false, now),
		},
		{
			on:   "two months ago",
			good: timeonly(false, twomonthsago),
			bad:  timeonly(false, now),
		},
		{
			on:   "four months ago",
			good: timeonly(false, fourmonthsago),
			bad:  timeonly(false, now),
		},
		{
			on:   "one year ago",
			good: timeonly(false, oneyearago),
			bad:  timeonly(false, now),
		},
		{
			on:   "four years ago",
			good: timeonly(false, fouryearsago),
			bad:  timeonly(false, now),
		},
	})
}

func TestFilterAfter(t *testing.T) {
	testmatch(t, []testholder{
		{
			after: "2020-02-14",
			good:  timeonly(false, time.Date(2024, 3, 14, 12, 0, 0, 0, time.Local), now, yesterday),
			bad:   timeonly(false, time.Date(2018, 2, 14, 12, 0, 0, 0, time.Local)),
		},
		{
			after: "yesterday",
			good:  timeonly(false, yesterday, yesterday.AddDate(1, 0, 0), now, now.AddDate(0, 3, 0)),
			bad:   timeonly(false, yesterday.AddDate(-1, 0, 0), yesterday.AddDate(0, 0, -1), ereyesterday),
		},
		{
			after: "one week ago",
			good:  timeonly(false, now),
			bad:   timeonly(false, oneweekago.AddDate(0, 0, -1)),
		},
		{
			after: "one month ago",
			good:  timeonly(false, now, oneweekago, twoweeksago),
			bad:   timeonly(false, onemonthago, twomonthsago, fourmonthsago, oneyearago),
		},
		{
			after: "two months ago",
			good:  timeonly(false, now, onemonthago, oneweekago),
			bad:   timeonly(false, twomonthsago, oneyearago, fourmonthsago),
		},
		{
			after: "four months ago",
			good:  timeonly(false, now, oneweekago, onemonthago, twoweeksago, twomonthsago, onemonthago),
			bad:   timeonly(false, fourmonthsago, oneyearago),
		},
		{
			after: "one year ago",
			good:  timeonly(false, now, onemonthago, twomonthsago, fourmonthsago),
			bad:   timeonly(false, oneyearago, fouryearsago, twoyearsago),
		},
		{
			after: "four years ago",
			good:  timeonly(false, now, twoyearsago, onemonthago, fourmonthsago),
			bad:   timeonly(false, fouryearsago, fouryearsago.AddDate(-1, 0, 0)),
		},
	})
}

func TestFilterBefore(t *testing.T) {
	testmatch(t, []testholder{
		{
			before: "2024-02-14",
			good:   timeonly(false, time.Date(2020, 2, 14, 12, 0, 0, 0, time.Local), time.Date(1989, 8, 13, 18, 53, 0, 0, time.Local)),
			bad:    timeonly(false, now, now.AddDate(0, 0, 10), now.AddDate(0, -2, 0)),
		},
		{
			before: "yesterday",
			good:   timeonly(false, onemonthago, oneweekago, oneyearago),
			bad:    timeonly(false, now, now.AddDate(0, 0, 1)),
		},
		{
			before: "one week ago",
			good:   timeonly(false, onemonthago, oneyearago, twoweeksago),
			bad:    timeonly(false, yesterday, now),
		},
		{
			before: "one month ago",
			good:   timeonly(false, oneyearago, twomonthsago),
			bad:    timeonly(false, oneweekago, yesterday, now),
		},
		{
			before: "two months ago",
			good:   timeonly(false, fourmonthsago, oneyearago),
			bad:    timeonly(false, onemonthago, oneweekago, yesterday, now),
		},
		{
			before: "four months ago",
			good:   timeonly(false, oneyearago, twoyearsago, fouryearsago),
			bad:    timeonly(false, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
		{
			before: "one year ago",
			good:   timeonly(false, twoyearsago, fouryearsago),
			bad:    timeonly(false, fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
		{
			before: "four years ago",
			good:   timeonly(false, fouryearsago.AddDate(-1, 0, 0), fouryearsago.AddDate(-4, 0, 0)),
			bad:    timeonly(false, oneyearago, fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
	})
}

func TestFilterMatch(t *testing.T) {
	testmatch(t, []testholder{
		{
			pattern: "[Tt]est",
			good:    nameonly(false, "test", "Test"),
			bad:     nameonly(false, "TEST", "tEst", "tEST", "TEst"),
		},
		{
			pattern: "^h.*o$",
			good:    nameonly(false, "hello", "hippo", "how about some pasta with alfredo"),
			bad:     nameonly(false, "hi", "test", "hellO", "Hello", "oh hello there"),
		},
	})
}

func TestFilterGlob(t *testing.T) {
	testmatch(t, []testholder{
		{
			glob: "*.txt",
			good: nameonly(false, "test.txt", "alsotest.txt"),
			bad:  nameonly(false, "test.md", "test.go", "test.tar.gz", "testxt", "test.text"),
		},
		{
			glob: "*.tar.*",
			good: nameonly(false, "test.tar.gz", "test.tar.xz", "test.tar.zst", "test.tar.bz2"),
			bad:  nameonly(false, "test.tar", "test.txt", "test.targz", "test.tgz"),
		},
		{
			glob: "pot*o",
			good: nameonly(false, "potato", "potdonkeyo", "potesto"),
			bad:  nameonly(false, "salad", "test", "alsotest"),
		},
		{
			glob: "t?st",
			good: nameonly(false, "test", "tist", "tfst", "tnst"),
			bad:  nameonly(false, "best", "fast", "most", "past"),
		},
	})
}

func TestFilterUnMatch(t *testing.T) {
	testmatch(t, []testholder{
		{
			unpattern: "^ss_.*\\.zip",
			good:      nameonly(false, "hello.zip", "ss_potato.png", "sss.zip"),
			bad:       nameonly(false, "ss_ost_flac.zip", "ss_guide.zip", "ss_controls.zip"),
		},
		{
			unpattern: "^h.*o$",
			good:      nameonly(false, "hi", "test", "hellO", "Hello", "oh hello there"),
			bad:       nameonly(false, "hello", "hippo", "how about some pasta with alfredo"),
		},
	})
}

func TestFilterUnGlob(t *testing.T) {
	testmatch(t, []testholder{
		{
			unglob: "*.txt",
			good:   nameonly(false, "test.md", "test.go", "test.tar.gz", "testxt", "test.text"),
			bad:    nameonly(false, "test.txt", "alsotest.txt"),
		},
		{
			unglob: "*.tar.*",
			good:   nameonly(false, "test.tar", "test.txt", "test.targz", "test.tgz"),
			bad:    nameonly(false, "test.tar.gz", "test.tar.xz", "test.tar.zst", "test.tar.bz2"),
		},
		{
			unglob: "pot*o",
			good:   nameonly(false, "salad", "test", "alsotest"),
			bad:    nameonly(false, "potato", "potdonkeyo", "potesto"),
		},
		{
			unglob: "t?st",
			good:   nameonly(false, "best", "fast", "most", "past"),
			bad:    nameonly(false, "test", "tist", "tfst", "tnst"),
		},
	})
}

func TestFilterFilenames(t *testing.T) {
	testmatch(t, []testholder{
		{
			filenames: []string{"test.txt", "alsotest.txt"},
			good:      nameonly(false, "test.txt", "alsotest.txt"),
			bad:       nameonly(false, "test.md", "test.go", "test.tar.gz", "testxt", "test.text"),
		},
		{
			filenames: []string{"test.md", "test.txt"},
			good:      nameonly(false, "test.txt", "test.md"),
			bad:       nameonly(false, "alsotest.txt", "test.go", "test.tar.gz", "testxt", "test.text"),
		},
		{
			filenames: []string{"hello.world"},
			good:      nameonly(false, "hello.world"),
			bad:       nameonly(false, "test.md", "test.go", "test.tar.gz", "testxt", "test.text", "helloworld", "Hello.world"),
		},
	})
}

func TestFilterFilesOnly(t *testing.T) {
	testmatch(t, []testholder{
		{
			filesonly: true,
			good:      nameonly(false, "test", "hellowold.txt", "test.md", "test.jpg"),
			bad:       nameonly(true, "test", "alsotest", "helloworld"),
		},
	})
}

func TestFilterDirsOnly(t *testing.T) {
	testmatch(t, []testholder{
		{
			dirsonly: true,
			good:     nameonly(true, "test", "alsotest", "helloworld"),
			bad:      nameonly(false, "test", "hellowold.txt", "test.md", "test.jpg"),
		},
		{
			dirsonly: true,
			good:     timeonly(true, fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
			bad:      timeonly(false, fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
	})
}

func TestFilterShowHidden(t *testing.T) {
	testmatch(t, []testholder{
		{
			ignorehidden: true,
			good:         append(nameonly(true, "test", "alsotest", "helloworld"), nameonly(false, "test", "alsotest", "helloworld")...),
			bad:          append(nameonly(true, ".test", ".alsotest", ".helloworld"), nameonly(false, ".test", ".alsotest", ".helloworld")...),
		},
		{
			ignorehidden: false,
			good:         append(nameonly(true, "test", "alsotest", ".helloworld"), nameonly(false, "test", "alsotest", ".helloworld")...),
		},
	})
}

func TestFilesize(t *testing.T) {
	testmatch(t, []testholder{
		{
			minsize: "9001B",
			good:    sizeonly(10000, 9002, 424242, math.MaxInt64),
			bad:     sizeonly(9000, math.MinInt64, 0, -9001),
		},
		{
			maxsize: "9001B",
			good:    sizeonly(9000, math.MinInt64, 0, -9001),
			bad:     sizeonly(10000, 9002, 424242, math.MaxInt64),
		},
	})
}

func TestMode(t *testing.T) {
	testmatch(t, []testholder{
		{
			mode: fs.FileMode(0755),
			good: modeonly(false, fs.FileMode(0755)),
			bad:  modeonly(false, fs.FileMode(0644)),
		},
	})
}

func TestFilterMultipleParameters(t *testing.T) {
	y, m, d := now.Date()
	threepm := time.Date(y, m, d, 15, 0, 0, 0, time.Local)
	tenpm := time.Date(y, m, d, 22, 0, 0, 0, time.Local)
	twoam := time.Date(y, m, d, 2, 0, 0, 0, time.Local)
	sevenam := time.Date(y, m, d, 7, 0, 0, 0, time.Local)

	testmatch(t, []testholder{
		{
			pattern: "[Tt]est",
			before:  "yesterday",
			good: []singletest{
				{filename: "test", modified: oneweekago},
				{filename: "test", modified: twoweeksago},
				{filename: "Test", modified: onemonthago},
				{filename: "Test", modified: fourmonthsago},
			},
			bad: []singletest{
				{filename: "test", modified: now},
				{filename: "salad", modified: oneweekago},
				{filename: "holyshit", modified: onemonthago},
			},
		},
		{
			glob:   "*.tar.*",
			before: "yesterday",
			after:  "one month ago",
			good: []singletest{
				{filename: "test.tar.xz", modified: oneweekago},
				{filename: "test.tar.gz", modified: twoweeksago},
				{filename: "test.tar.zst", modified: twoweeksago.AddDate(0, 0, 2)},
				{filename: "test.tar.bz2", modified: twoweeksago.AddDate(0, 0, -4)},
			},
			bad: []singletest{
				{filename: "test.tar.gz", modified: oneyearago},
				{filename: "test.targz", modified: oneweekago},
				{filename: "test.jpg", modified: ereyesterday},
			},
		},
		{
			on:     "today",
			after:  "two weeks ago",
			before: "one week ago",
			good:   timeonly(false, now, twoam, sevenam, threepm, tenpm),
			bad:    timeonly(false, yesterday, oneweekago, onemonthago, oneyearago),
		},
		{
			unpattern: ".*\\.(jpg|png)",
			on:        "today",
			good: []singletest{
				{filename: "test.txt", modified: now},
				{filename: "hello.md", modified: tenpm},
			},
			bad: []singletest{
				{filename: "test.png", modified: now},
				{filename: "test.jpg", modified: twoam},
				{filename: "hello.md", modified: twomonthsago},
			},
		},
		{
			filesonly: true,
			unglob:    "*.txt",
			good:      nameonly(false, "test.md", "test.jpg", "test.png"),
			bad: []singletest{
				{
					filename: "test",
					isdir:    true,
				},
				{
					filename: "test.txt",
					isdir:    false,
				},
				{
					filename: "test.md",
					isdir:    true,
				},
			},
		},
		{
			dirsonly: true,
			pattern:  "w(or|ea)ld",
			good:     nameonly(true, "hello world", "high weald"),
			bad: []singletest{
				{
					filename: "hello_world.txt",
					isdir:    false,
				},
				{
					filename: "highweald.txt",
					isdir:    false,
				},
			},
		},
		{
			glob:    "*.txt",
			minsize: "4096B",
			good: []singletest{
				{
					filename: "test.txt",
					size:     9001,
				},
				{
					filename: "hello.txt",
					size:     4097,
				},
			},
			bad: []singletest{
				{
					filename: "test.md",
					size:     9001,
				},
				{
					filename: "test.txt",
					size:     1024,
				},
			},
		},
		{
			mode:         fs.FileMode(0600),
			ignorehidden: true,
			good: []singletest{
				{
					mode:     fs.FileMode(0600),
					filename: "hello.txt",
				},
				{
					mode:     fs.FileMode(0600),
					filename: "main.go",
				},
			},
			bad: []singletest{
				{
					mode:     fs.FileMode(0600),
					filename: ".bashrc",
				},
				{
					mode:     fs.FileMode(0644),
					filename: "hello.txt",
				},
				{
					mode:     fs.FileMode(0644),
					filename: "main.go",
				},
			},
		},
	})
}

func TestFilterBlank(t *testing.T) {
	var fltr *filter.Filter
	t.Run("new", func(t *testing.T) {
		fltr, _ = filter.New("", "", "", "", "", "", "", false, false, false, "0", "0", 0)
		if !fltr.Blank() {
			t.Fatalf("filter isn't blank? %s", fltr)
		}
	})

	t.Run("blank", func(t *testing.T) {
		fltr = &filter.Filter{}
		if !fltr.Blank() {
			t.Fatalf("filter isn't blank? %s", fltr)
		}
	})
}

func TestFilterNotBlank(t *testing.T) {
	var (
		fltr    *filter.Filter
		testers = []testholder{
			{
				pattern: "[Ttest]",
			},
			{
				glob: "*test*",
			},
			{
				unpattern: ".*\\.(jpg|png)",
			},
			{
				unglob: "*.jpg",
			},
			{
				before: "yesterday",
				after:  "one week ago",
			},
			{
				on: "2024-06-06",
			},
			{
				filenames: []string{"hello"},
			},
			{
				filenames: []string{""},
			},
			{
				filesonly: true,
			},
			{
				dirsonly: true,
			},
			{
				ignorehidden: true,
			},
			{
				mode: fs.FileMode(0644),
			},
		}
	)

	for _, tester := range testers {
		t.Run("notblank"+tester.String(), func(t *testing.T) {
			fltr, _ = filter.New(
				tester.on, tester.before, tester.after, tester.glob, tester.pattern,
				tester.unglob, tester.unpattern, tester.filesonly, tester.dirsonly,
				tester.ignorehidden, tester.minsize, tester.maxsize, tester.mode,
				tester.filenames...,
			)
			if fltr.Blank() {
				t.Fatalf("filter is blank?? %s", fltr)
			}
		})
	}
}
