package filter

import (
	"fmt"
	"testing"
	"time"
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
}

func (t testholder) String() string {
	return fmt.Sprintf(
		"pattern:'%s' glob:'%s' unpattern:'%s' unglob:'%s' filenames:'%v' "+
			"before:'%s' after:'%s' on:'%s' filesonly:'%t' dirsonly:'%t' ignorehidden:'%t'",
		t.pattern, t.glob, t.unpattern, t.unglob, t.filenames, t.before, t.after, t.on,
		t.filesonly, t.dirsonly, t.ignorehidden,
	)
}

type singletest struct {
	filename string
	isdir    bool
	modified time.Time
}

func (s singletest) String() string {
	return fmt.Sprintf("filename:'%s' modified:'%s' isdir:'%t'", s.filename, s.modified, s.isdir)
}

func testmatch(t *testing.T, testers []testholder) {
	const testnamefmt string = "file %s modified on %s"
	var (
		f   *Filter
		err error
	)
	for _, tester := range testers {
		f, err = New(
			tester.on, tester.before, tester.after, tester.glob, tester.pattern,
			tester.unglob, tester.unpattern, tester.filesonly, tester.dirsonly, tester.ignorehidden,
			tester.filenames...,
		)
		if err != nil {
			t.Fatal(err)
		}

		for _, tst := range tester.good {
			t.Run(fmt.Sprintf(testnamefmt+"_good", tst.filename, tst.modified), func(t *testing.T) {
				if !f.Match(tst.filename, tst.modified, tst.isdir) {
					t.Fatalf("(filename:%s modified:%s) didn't match (%s) but should have", tst.filename, tst.modified, tester)
				}
			})
		}

		for _, tst := range tester.bad {
			t.Run(fmt.Sprintf(testnamefmt+"_bad", tst.filename, tst.modified), func(t *testing.T) {
				if f.Match(tst.filename, tst.modified, tst.isdir) {
					t.Fatalf("(filename:%s modified:%s) matched (%s) but shouldn't have", tst.filename, tst.modified, tester)
				}
			})
		}
	}
}

func blankfilename(times ...time.Time) []singletest {
	out := make([]singletest, 0, len(times))
	for _, time := range times {
		out = append(out, singletest{filename: "blank.txt", modified: time, isdir: false})
	}
	return out
}

func blankdirname(times ...time.Time) []singletest {
	out := make([]singletest, 0, len(times))
	for _, time := range times {
		out = append(out, singletest{filename: "blank", modified: time, isdir: true})
	}
	return out
}

func blanktimefile(filenames ...string) []singletest {
	out := make([]singletest, 0, len(filenames))
	for _, filename := range filenames {
		out = append(out, singletest{filename: filename, modified: time.Time{}, isdir: false})
	}
	return out
}

func blanktimedir(dirnames ...string) []singletest {
	out := make([]singletest, 0, len(dirnames))
	for _, dirname := range dirnames {
		out = append(out, singletest{filename: dirname, modified: time.Time{}, isdir: true})
	}
	return out
}

func TestFilterOn(t *testing.T) {
	testmatch(t, []testholder{
		{
			on:   "2024-02-14",
			good: blankfilename(time.Date(2024, 2, 14, 12, 0, 0, 0, time.Local)),
			bad:  blankfilename(now, now.Add(time.Hour*72), now.Add(-time.Hour*18)),
		},
		{
			on:   "yesterday",
			good: blankfilename(yesterday),
			bad:  blankfilename(now, oneweekago, onemonthago, oneyearago, twoweeksago, twomonthsago, twoyearsago),
		},
		{
			on:   "one week ago",
			good: blankfilename(oneweekago),
			bad:  blankfilename(now),
		},
		{
			on:   "one month ago",
			good: blankfilename(onemonthago),
			bad:  blankfilename(now),
		},
		{
			on:   "two months ago",
			good: blankfilename(twomonthsago),
			bad:  blankfilename(now),
		},
		{
			on:   "four months ago",
			good: blankfilename(fourmonthsago),
			bad:  blankfilename(now),
		},
		{
			on:   "one year ago",
			good: blankfilename(oneyearago),
			bad:  blankfilename(now),
		},
		{
			on:   "four years ago",
			good: blankfilename(fouryearsago),
			bad:  blankfilename(now),
		},
	})
}

func TestFilterAfter(t *testing.T) {
	testmatch(t, []testholder{
		{
			after: "2020-02-14",
			good:  blankfilename(time.Date(2024, 3, 14, 12, 0, 0, 0, time.Local), now, yesterday),
			bad:   blankfilename(time.Date(2018, 2, 14, 12, 0, 0, 0, time.Local)),
		},
		{
			after: "yesterday",
			good:  blankfilename(yesterday, yesterday.AddDate(1, 0, 0), now, now.AddDate(0, 3, 0)),
			bad:   blankfilename(yesterday.AddDate(-1, 0, 0), yesterday.AddDate(0, 0, -1), ereyesterday),
		},
		{
			after: "one week ago",
			good:  blankfilename(now),
			bad:   blankfilename(oneweekago.AddDate(0, 0, -1)),
		},
		{
			after: "one month ago",
			good:  blankfilename(now, oneweekago, twoweeksago),
			bad:   blankfilename(onemonthago, twomonthsago, fourmonthsago, oneyearago),
		},
		{
			after: "two months ago",
			good:  blankfilename(now, onemonthago, oneweekago),
			bad:   blankfilename(twomonthsago, oneyearago, fourmonthsago),
		},
		{
			after: "four months ago",
			good:  blankfilename(now, oneweekago, onemonthago, twoweeksago, twomonthsago, onemonthago),
			bad:   blankfilename(fourmonthsago, oneyearago),
		},
		{
			after: "one year ago",
			good:  blankfilename(now, onemonthago, twomonthsago, fourmonthsago),
			bad:   blankfilename(oneyearago, fouryearsago, twoyearsago),
		},
		{
			after: "four years ago",
			good:  blankfilename(now, twoyearsago, onemonthago, fourmonthsago),
			bad:   blankfilename(fouryearsago, fouryearsago.AddDate(-1, 0, 0)),
		},
	})
}

func TestFilterBefore(t *testing.T) {
	testmatch(t, []testholder{
		{
			before: "2024-02-14",
			good:   blankfilename(time.Date(2020, 2, 14, 12, 0, 0, 0, time.Local), time.Date(1989, 8, 13, 18, 53, 0, 0, time.Local)),
			bad:    blankfilename(now, now.AddDate(0, 0, 10), now.AddDate(0, -2, 0)),
		},
		{
			before: "yesterday",
			good:   blankfilename(onemonthago, oneweekago, oneyearago),
			bad:    blankfilename(now, now.AddDate(0, 0, 1)),
		},
		{
			before: "one week ago",
			good:   blankfilename(onemonthago, oneyearago, twoweeksago),
			bad:    blankfilename(yesterday, now),
		},
		{
			before: "one month ago",
			good:   blankfilename(oneyearago, twomonthsago),
			bad:    blankfilename(oneweekago, yesterday, now),
		},
		{
			before: "two months ago",
			good:   blankfilename(fourmonthsago, oneyearago),
			bad:    blankfilename(onemonthago, oneweekago, yesterday, now),
		},
		{
			before: "four months ago",
			good:   blankfilename(oneyearago, twoyearsago, fouryearsago),
			bad:    blankfilename(twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
		{
			before: "one year ago",
			good:   blankfilename(twoyearsago, fouryearsago),
			bad:    blankfilename(fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
		{
			before: "four years ago",
			good:   blankfilename(fouryearsago.AddDate(-1, 0, 0), fouryearsago.AddDate(-4, 0, 0)),
			bad:    blankfilename(oneyearago, fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
	})
}

func TestFilterMatch(t *testing.T) {
	testmatch(t, []testholder{
		{
			pattern: "[Tt]est",
			good:    blanktimefile("test", "Test"),
			bad:     blanktimefile("TEST", "tEst", "tEST", "TEst"),
		},
		{
			pattern: "^h.*o$",
			good:    blanktimefile("hello", "hippo", "how about some pasta with alfredo"),
			bad:     blanktimefile("hi", "test", "hellO", "Hello", "oh hello there"),
		},
	})
}

func TestFilterGlob(t *testing.T) {
	testmatch(t, []testholder{
		{
			glob: "*.txt",
			good: blanktimefile("test.txt", "alsotest.txt"),
			bad:  blanktimefile("test.md", "test.go", "test.tar.gz", "testxt", "test.text"),
		},
		{
			glob: "*.tar.*",
			good: blanktimefile("test.tar.gz", "test.tar.xz", "test.tar.zst", "test.tar.bz2"),
			bad:  blanktimefile("test.tar", "test.txt", "test.targz", "test.tgz"),
		},
		{
			glob: "pot*o",
			good: blanktimefile("potato", "potdonkeyo", "potesto"),
			bad:  blanktimefile("salad", "test", "alsotest"),
		},
		{
			glob: "t?st",
			good: blanktimefile("test", "tast", "tfst", "tnst"),
			bad:  blanktimefile("best", "fast", "most", "past"),
		},
	})
}

func TestFilterUnMatch(t *testing.T) {
	testmatch(t, []testholder{
		{
			unpattern: "^ss_.*\\.zip",
			good:      blanktimefile("hello.zip", "ss_potato.png", "sss.zip"),
			bad:       blanktimefile("ss_ost_flac.zip", "ss_guide.zip", "ss_controls.zip"),
		},
		{
			unpattern: "^h.*o$",
			good:      blanktimefile("hi", "test", "hellO", "Hello", "oh hello there"),
			bad:       blanktimefile("hello", "hippo", "how about some pasta with alfredo"),
		},
	})
}

func TestFilterUnGlob(t *testing.T) {
	testmatch(t, []testholder{
		{
			unglob: "*.txt",
			good:   blanktimefile("test.md", "test.go", "test.tar.gz", "testxt", "test.text"),
			bad:    blanktimefile("test.txt", "alsotest.txt"),
		},
		{
			unglob: "*.tar.*",
			good:   blanktimefile("test.tar", "test.txt", "test.targz", "test.tgz"),
			bad:    blanktimefile("test.tar.gz", "test.tar.xz", "test.tar.zst", "test.tar.bz2"),
		},
		{
			unglob: "pot*o",
			good:   blanktimefile("salad", "test", "alsotest"),
			bad:    blanktimefile("potato", "potdonkeyo", "potesto"),
		},
		{
			unglob: "t?st",
			good:   blanktimefile("best", "fast", "most", "past"),
			bad:    blanktimefile("test", "tast", "tfst", "tnst"),
		},
	})
}

func TestFilterFilenames(t *testing.T) {
	testmatch(t, []testholder{
		{
			filenames: []string{"test.txt", "alsotest.txt"},
			good:      blanktimefile("test.txt", "alsotest.txt"),
			bad:       blanktimefile("test.md", "test.go", "test.tar.gz", "testxt", "test.text"),
		},
		{
			filenames: []string{"test.md", "test.txt"},
			good:      blanktimefile("test.txt", "test.md"),
			bad:       blanktimefile("alsotest.txt", "test.go", "test.tar.gz", "testxt", "test.text"),
		},
		{
			filenames: []string{"hello.world"},
			good:      blanktimefile("hello.world"),
			bad:       blanktimefile("test.md", "test.go", "test.tar.gz", "testxt", "test.text", "helloworld", "Hello.world"),
		},
	})
}

func TestFilterFilesOnly(t *testing.T) {
	testmatch(t, []testholder{
		{
			filesonly: true,
			good:      blanktimefile("test", "hellowold.txt", "test.md", "test.jpg"),
			bad:       blanktimedir("test", "alsotest", "helloworld"),
		},
	})
}

func TestFilterDirsOnly(t *testing.T) {
	testmatch(t, []testholder{
		{
			dirsonly: true,
			good:     blanktimedir("test", "alsotest", "helloworld"),
			bad:      blanktimefile("test", "hellowold.txt", "test.md", "test.jpg"),
		},
		{
			dirsonly: true,
			good:     blankdirname(fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
			bad:      blankfilename(fourmonthsago, twomonthsago, onemonthago, oneweekago, yesterday, now),
		},
	})
}

func TestFilterIgnoreHidden(t *testing.T) {
	testmatch(t, []testholder{
		{
			ignorehidden: true,
			good:         append(blanktimedir("test", "alsotest", "helloworld"), blanktimefile("test", "alsotest", "helloworld")...),
			bad:          append(blanktimedir(".test", ".alsotest", ".helloworld"), blanktimefile(".test", ".alsotest", ".helloworld")...),
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
			good:   blankfilename(now, twoam, sevenam, threepm, tenpm),
			bad:    blankfilename(yesterday, oneweekago, onemonthago, oneyearago),
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
			good:      blanktimefile("test.md", "test.jpg", "test.png"),
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
			good:     blanktimedir("hello world", "high weald"),
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
	})
}

func TestFilterBlank(t *testing.T) {
	var f *Filter
	t.Run("new", func(t *testing.T) {
		f, _ = New("", "", "", "", "", "", "", false, false, false)
		if !f.Blank() {
			t.Fatalf("filter isn't blank? %s", f)
		}
	})

	t.Run("blank", func(t *testing.T) {
		f = &Filter{}
		if !f.Blank() {
			t.Fatalf("filter isn't blank? %s", f)
		}
	})
}

func TestFilterNotBlank(t *testing.T) {
	var (
		f       *Filter
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
		}
	)

	for _, tester := range testers {
		t.Run("notblank"+tester.String(), func(t *testing.T) {
			f, _ = New(
				tester.on, tester.before, tester.after, tester.glob, tester.pattern,
				tester.unglob, tester.unpattern, tester.filesonly, tester.dirsonly, tester.ignorehidden,
				tester.filenames...,
			)
			if f.Blank() {
				t.Fatalf("filter is blank?? %s", f)
			}
		})
	}
}
