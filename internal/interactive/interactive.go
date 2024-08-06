// Package interactive implements a charm-powered table to display files.
package interactive

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"git.burning.moe/celediel/gt/internal/dirs"
	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/interactive/modes"
	"git.burning.moe/celediel/gt/internal/interactive/sorting"
	"golang.org/x/term"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

const (
	uncheck string = "☐"
	check   string = "☑"
	space   string = " "
	woffset int    = 13 // why this number, I don't know
	hoffset int    = 6
	poffset int    = 2

	filenameColumn string = "filename"
	pathColumn     string = "path"
	modifiedColumn string = "modified"
	trashedColumn  string = "trashed"
	sizeColumn     string = "size"
	bar            string = "───"

	// TODO: figure these out dynamically based on longest of each
	filenameColumnW float64 = 0.46
	pathColumnW     float64 = 0.25
	dateColumnW     float64 = 0.15
	sizeColumnW     float64 = 0.12
	checkColumnW    float64 = 0.02

	// TODO: make these configurable or something
	borderbg    string = "5"
	hoveritembg string = "13"
	black       string = "0"
	darkblack   string = "8"
	white       string = "7"
	darkgray    string = "15"
)

var (
	style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderbg))
	regulartext = lipgloss.NewStyle().
			Padding(0, poffset)
	darktext = lipgloss.NewStyle().
			Foreground(lipgloss.Color(darkgray))
	darkertext = lipgloss.NewStyle().
			Foreground(lipgloss.Color(darkblack))
	darkesttext = lipgloss.NewStyle().
			Foreground(lipgloss.Color(black))
)

type model struct {
	table      table.Model
	keys       keyMap
	selected   map[string]bool
	selectsize int64
	readonly   bool
	once       bool
	filtering  bool
	filter     string
	termheight int
	termwidth  int
	mode       modes.Mode
	sorting    sorting.Sorting
	workdir    string
	files      files.Files
	fltrfiles  files.Files
}

func newModel(fls []files.File, selectall, readonly, once bool, workdir string, mode modes.Mode) model {
	m := model{
		keys:       defaultKeyMap(),
		readonly:   readonly,
		once:       once,
		mode:       mode,
		selected:   map[string]bool{},
		selectsize: 0,
		files:      fls,
	}

	m.termwidth, m.termheight = termSizes()

	if workdir != "" {
		m.workdir = filepath.Clean(workdir)
	}

	rows := m.freshRows()
	columns := m.freshColumns()

	theight := min(m.termheight-hoffset, len(fls))
	m.table = createTable(columns, rows, theight)

	m.sorting = sorting.Name
	m.sort()

	if selectall {
		m.selectAll()
	}

	return m
}

type keyMap struct {
	mark key.Binding
	doit key.Binding
	todo key.Binding
	nada key.Binding
	invr key.Binding
	rstr key.Binding
	clen key.Binding
	sort key.Binding
	rort key.Binding
	fltr key.Binding
	clfl key.Binding
	apfl key.Binding
	bksp key.Binding
	quit key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		mark: key.NewBinding(
			key.WithKeys(space),
			key.WithHelp("space", "toggle"),
		),
		doit: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "confirm"),
		),
		todo: key.NewBinding(
			key.WithKeys("a", "ctrl+a"),
			key.WithHelp("a", "all"),
		),
		nada: key.NewBinding(
			key.WithKeys("n", "ctrl+n"),
			key.WithHelp("n", "none"),
		),
		invr: key.NewBinding(
			key.WithKeys("i", "ctrl+i"),
			key.WithHelp("i", "invert"),
		),
		clen: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clean"),
		),
		rstr: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restore"),
		),
		sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s/S", "sort"),
		),
		rort: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "sort (reverse)"),
		),
		fltr: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		apfl: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
		clfl: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		bksp: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "backspace"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (m model) Init() tea.Cmd {
	/* if m.onePage() {
		m.table.SetStyles(makeUnselectedStyle())
		m.unselectAll()
		return tea.Quit
	} */
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.once {
		return m.quit(m.readonly)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.updateTableSize()
	case tea.KeyMsg:
		if m.filtering {
			switch {
			case key.Matches(msg, m.keys.clfl):
				m.filter = ""
				m.filtering = false
			case key.Matches(msg, m.keys.apfl):
				m.filtering = false
			case key.Matches(msg, m.keys.bksp):
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
				}
			default:
				m.filter += msg.String()
			}
			m.applyFilter()
			return m, cmd
		}

		switch {
		case key.Matches(msg, m.keys.mark):
			m.toggleItem(m.table.Cursor())
		case key.Matches(msg, m.keys.doit):
			if !m.readonly && m.mode != modes.Interactive && len(m.fltrfiles) > 1 {
				return m.quit(false)
			}
		case key.Matches(msg, m.keys.nada):
			m.unselectAll()
		case key.Matches(msg, m.keys.todo):
			m.selectAll()
		case key.Matches(msg, m.keys.invr):
			m.invertSelection()
		case key.Matches(msg, m.keys.clen):
			return m.execute(modes.Cleaning)
		case key.Matches(msg, m.keys.rstr):
			return m.execute(modes.Restoring)
		case key.Matches(msg, m.keys.sort):
			m.sorting = m.sorting.Next()
			m.sort()
		case key.Matches(msg, m.keys.rort):
			m.sorting = m.sorting.Prev()
			m.sort()
		case key.Matches(msg, m.keys.fltr):
			m.filtering = true
		case key.Matches(msg, m.keys.clfl):
			if m.filter != "" {
				m.filter = ""
				m.applyFilter()
			}
		case key.Matches(msg, m.keys.quit):
			return m.quit(true)
		}
	}

	// pass events along to the table
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var panels []string
	if !m.once {
		panels = append(panels, m.header())
	}

	panels = append(panels, style.Render(m.table.View()), m.footer())

	return lipgloss.JoinVertical(lipgloss.Top,
		panels...,
	)
}

func (m model) showHelp() string {
	var filterText string
	if m.filter != "" {
		filterText = fmt.Sprintf(" (%s)", m.filter)
	}

	keys := []string{
		fmt.Sprintf("%s %s%s", darktext.Render(m.keys.fltr.Help().Key), darkertext.Render(m.keys.fltr.Help().Desc), filterText),
		fmt.Sprintf("%s %s (%s)", darktext.Render(m.keys.sort.Help().Key), darkertext.Render(m.keys.sort.Help().Desc), m.sorting.String()),
		styleKey(m.keys.quit),
	}

	if !m.readonly {
		if m.mode != modes.Interactive {
			keys = append([]string{styleKey(m.keys.doit)}, keys...)
		}
		keys = append([]string{styleKey(m.keys.mark)}, keys...)
	}
	return strings.Join(keys, darkesttext.Render(" • "))
}

func (m model) header() string {
	var (
		right, left string
		spacerWidth int
		keys        = []string{
			styleKey(m.keys.rstr),
			styleKey(m.keys.clen),
		}
		selectKeys = []string{
			styleKey(m.keys.todo),
			styleKey(m.keys.nada),
			styleKey(m.keys.invr),
		}
		filterKeys = []string{
			styleKey(m.keys.clfl),
			styleKey(m.keys.apfl),
		}
		dot       = darkesttext.Render("•")
		wideDot   = darkesttext.Render(" • ")
		keysFmt   = strings.Join(keys, wideDot)
		selectFmt = strings.Join(selectKeys, wideDot)
		filterFmt = strings.Join(filterKeys, wideDot)
	)

	switch {
	case m.filtering:
		right = fmt.Sprintf(" Filtering %s %s", dot, filterFmt)
	case m.mode == modes.Interactive:
		right = fmt.Sprintf(" %s %s %s", keysFmt, dot, selectFmt)
		left = fmt.Sprintf("%d/%d %s %s", len(m.selected), len(m.fltrfiles), dot, humanize.Bytes(uint64(m.selectsize)))
	case m.mode == modes.Listing:
		var filtered string
		if m.filter != "" || m.filtering {
			filtered = " (filtered)"
		}
		right = fmt.Sprintf(" Showing%s %d files in trash", filtered, len(m.fltrfiles))
	default:
		var wd string
		if m.workdir != "" {
			wd = " in " + dirs.UnExpand(m.workdir, "")
		}
		right = fmt.Sprintf(" %s%s %s %s", m.mode.String(), wd, dot, selectFmt)
		left = fmt.Sprintf("%d/%d %s %s", len(m.selected), len(m.fltrfiles), dot, humanize.Bytes(uint64(m.selectsize)))
	}

	// offset of 2 again because of table border
	spacerWidth = m.termwidth - lipgloss.Width(right) - lipgloss.Width(left) - poffset
	if spacerWidth <= 0 {
		spacerWidth = 1 // always at least one space
	}

	return fmt.Sprintf("%s%s%s", right, strings.Repeat(" ", spacerWidth), left)
}

func (m model) footer() string {
	return regulartext.Render(m.showHelp())
}

func (m model) quit(unselectAll bool) (model, tea.Cmd) {
	if unselectAll {
		m.unselectAll()
	} else {
		m.onlySelected()
	}
	m.table.SetStyles(makeUnselectedStyle())
	return m, tea.Quit
}

func (m model) execute(mode modes.Mode) (model, tea.Cmd) {
	if m.mode != modes.Interactive || len(m.selected) <= 0 || len(m.fltrfiles) <= 0 {
		var cmd tea.Cmd
		return m, cmd
	}

	m.mode = mode
	m.onlySelected()
	m.table.SetStyles(makeUnselectedStyle())
	return m, tea.Quit
}

func (m model) selectedFiles() (outfile files.Files) {
	for _, file := range m.fltrfiles {
		if m.selected[file.String()] {
			outfile = append(outfile, file)
		}
	}
	return
}

/* func (m model) onePage() bool {
	x := m.termheight
	y := len(m.table.Rows()) + hoffset
	if x > y && m.readonly {
		return true
	}
	return false
} */

func (m *model) freshRows() (rows []table.Row) {
	for _, file := range m.files {
		row := newRow(file, m.workdir)

		if !m.readonly {
			row = append(row, getCheck(false))
		}
		rows = append(rows, row)
	}
	return
}

func (m *model) onlySelected() {
	var rows = make([]table.Row, 0)
	for _, row := range m.table.Rows() {
		if row[4] == check {
			rows = append(rows, row)
		} else {
			rows = append(rows, table.Row{})
		}
	}
	m.table.SetRows(rows)
}

// updateRow updates row of provided index with provided row.
func (m *model) updateRow(index int, selected bool) {
	rows := m.table.Rows()
	row := rows[index]
	rows[index] = table.Row{
		row[0],
		row[1],
		row[2],
		row[3],
		getCheck(selected),
	}

	m.table.SetRows(rows)
}

func (m *model) updateRows(selected bool) {
	var newrows = []table.Row{}

	for _, row := range m.table.Rows() {
		newRow := table.Row{
			row[0],
			row[1],
			row[2],
			row[3],
			getCheck(selected),
		}
		newrows = append(newrows, newRow)
	}
	m.table.SetRows(newrows)
}

func (m *model) toggleItem(index int) (selected bool) {
	if m.readonly || len(m.fltrfiles) == 0 {
		return false
	}

	name := m.fltrfiles[index].String()
	size := m.fltrfiles[index].Filesize()

	// select the thing
	if v, ok := m.selected[name]; v && ok {
		// already selected
		delete(m.selected, name)
		selected = false
		m.selectsize -= size
	} else {
		// not selected
		m.selected[name] = true
		selected = true
		m.selectsize += size
	}

	// update the rows with the state
	m.updateRow(index, selected)
	return
}

func (m *model) selectAll() {
	if m.readonly || len(m.fltrfiles) == 0 {
		return
	}

	m.selected = map[string]bool{}
	m.selectsize = 0
	for i := range m.table.Rows() {
		m.selected[m.fltrfiles[i].String()] = true
		m.selectsize += m.fltrfiles[i].Filesize()
	}
	m.updateRows(true)
}

func (m *model) unselectAll() {
	if m.readonly {
		return
	}

	m.selected = map[string]bool{}
	m.selectsize = 0
	m.updateRows(false)
}

func (m *model) invertSelection() {
	if m.readonly || len(m.fltrfiles) == 0 {
		return
	}

	var newrows []table.Row

	for index, row := range m.table.Rows() {
		name := m.fltrfiles[index].String()
		size := m.fltrfiles[index].Filesize()
		if v, ok := m.selected[name]; v && ok {
			delete(m.selected, name)
			m.selectsize -= size
			newrows = append(newrows, table.Row{
				row[0],
				row[1],
				row[2],
				row[3],
				getCheck(false),
			})
		} else {
			m.selected[name] = true
			m.selectsize += size
			newrows = append(newrows, table.Row{
				row[0],
				row[1],
				row[2],
				row[3],
				getCheck(true),
			})
		}
	}

	m.table.SetRows(newrows)
}

func (m *model) sort() {
	slices.SortStableFunc(m.files, m.sorting.Sorter())
	m.applyFilter()
}

func (m *model) applyFilter() {
	m.fltrfiles = m.filteredFiles()
	var rows = []table.Row{}
	for _, file := range m.fltrfiles {
		row := newRow(file, m.workdir)
		if !m.readonly {
			row = append(row, getCheck(m.selected[file.String()]))
		}
		rows = append(rows, row)
	}

	if len(rows) < 1 {
		row := table.Row{"no files matched filter!", bar, bar, bar}
		if !m.readonly {
			row = append(row, uncheck)
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
	m.updateTableHeight()
}

func (m *model) filteredFiles() (filteredFiles files.Files) {
	for _, file := range m.files {
		if isMatch(m.filter, file.Name()) {
			filteredFiles = append(filteredFiles, file)
		} else {
			if _, ok := m.selected[file.String()]; ok {
				delete(m.selected, file.String())
				m.selectsize -= file.Filesize()
			}
		}
	}
	return
}

func (m *model) freshColumns() []table.Column {
	var (
		fwidth     = int(math.Round(float64(m.termwidth-woffset) * filenameColumnW))
		owidth     = int(math.Round(float64(m.termwidth-woffset) * pathColumnW))
		dwidth     = int(math.Round(float64(m.termwidth-woffset) * dateColumnW))
		swidth     = int(math.Round(float64(m.termwidth-woffset) * sizeColumnW))
		cwidth     = int(math.Round(float64(m.termwidth-woffset) * checkColumnW))
		datecolumn string
	)

	switch m.mode {
	case modes.Trashing:
		datecolumn = modifiedColumn
	default:
		datecolumn = trashedColumn
	}

	columns := []table.Column{
		{Title: filenameColumn, Width: fwidth},
		{Title: pathColumn, Width: owidth},
		{Title: datecolumn, Width: dwidth},
		{Title: sizeColumn, Width: swidth},
	}

	if !m.readonly {
		columns = append(columns, table.Column{Title: uncheck, Width: cwidth})
	} else {
		columns[0].Width += cwidth
	}

	return columns
}

func (m *model) updateTableSize() {
	width, height := termSizes()
	m.termheight = height
	m.termwidth = width - poffset
	m.table.SetWidth(m.termwidth)
	m.updateTableHeight()
	m.table.SetColumns(m.freshColumns())
}

func (m *model) updateTableHeight() {
	h := min(m.termheight-hoffset, len(m.table.Rows()))
	m.table.SetHeight(h)
	if m.table.Cursor() >= h {
		m.table.SetCursor(h - 1)
	}
}

func Select(fls files.Files, selectall, once bool, workdir string, mode modes.Mode) (files.Files, modes.Mode, error) {
	mdl := newModel(fls, selectall, false, once, workdir, mode)
	endmodel, err := tea.NewProgram(mdl).Run()
	if err != nil {
		return fls, 0, err
	}
	m, ok := endmodel.(model)
	if !ok {
		return fls, 0, fmt.Errorf("model isn't the right type?? what has happened")
	}
	return m.selectedFiles(), m.mode, nil
}

func Show(fls files.Files, once bool, workdir string) error {
	mdl := newModel(fls, false, true, once, workdir, modes.Listing)
	if _, err := tea.NewProgram(mdl).Run(); err != nil {
		return err
	}
	return nil
}

func newRow(file files.File, workdir string) table.Row {
	var time, size string
	time = humanize.Time(file.Date())
	if file.IsDir() {
		size = bar
	} else {
		size = humanize.Bytes(uint64(file.Filesize()))
	}
	return table.Row{
		dirs.UnEscape(file.Name()),
		dirs.UnExpand(filepath.Dir(file.Path()), workdir),
		time,
		size,
	}
}

func getCheck(selected bool) (ourcheck string) {
	if selected {
		ourcheck = check
	} else {
		ourcheck = uncheck
	}
	return
}

func createTable(columns []table.Column, rows []table.Row, height int) table.Model {
	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	tbl.KeyMap = fixTableKeymap()
	tbl.SetStyles(makeStyle())
	return tbl
}

func fixTableKeymap() table.KeyMap {
	keys := table.DefaultKeyMap()

	// remove spacebar from default page down keybind, but keep the rest
	keys.PageDown.SetKeys(
		slices.DeleteFunc(keys.PageDown.Keys(), func(s string) bool {
			return s == space
		})...,
	)

	return keys
}

func makeStyle() table.Styles {
	style := table.DefaultStyles()
	style.Header = style.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(black)).
		BorderBottom(true).
		Bold(false)
	style.Selected = style.Selected.
		Foreground(lipgloss.Color(white)).
		Background(lipgloss.Color(hoveritembg)).
		Bold(false)

	return style
}

func makeUnselectedStyle() table.Styles {
	style := makeStyle()
	style.Selected = style.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)
	return style
}

func styleKey(key key.Binding) string {
	return fmt.Sprintf("%s %s", darktext.Render(key.Help().Key), darkertext.Render(key.Help().Desc))
}

func isMatch(pattern, filename string) bool {
	p := strings.ToLower(pattern)
	f := strings.ToLower(filename)
	return fuzzy.Match(p, f)
}

func termSizes() (width int, height int) {
	// read the term height and width for tables
	var err error
	width, height, err = term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80
		height = 24
	}
	return
}
