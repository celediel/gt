// Package interactive implements a charm-powered table to display files.
package interactive

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"git.burning.moe/celediel/gt/internal/dirs"
	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/interactive/modes"
	"git.burning.moe/celediel/gt/internal/interactive/sorting"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

const (
	uncheck string = "☐"
	check   string = "☑"
	space   string = " "
	woffset int    = 13 // why this number, I don't know
	hoffset int    = 6

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
			Padding(0, 2)
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
	termheight int
	termwidth  int
	mode       modes.Mode
	sorting    sorting.Sorting
	workdir    string
	files      files.Files
}

func newModel(fls []files.File, width, height int, readonly, preselected, once bool, workdir string, mode modes.Mode) model {
	var (
		// TODO: figure this out dynamically based on longest of each
		fwidth  int = int(math.Round(float64(width-woffset) * 0.46))
		owidth  int = int(math.Round(float64(width-woffset) * 0.25))
		dwidth  int = int(math.Round(float64(width-woffset) * 0.15))
		swidth  int = int(math.Round(float64(width-woffset) * 0.12))
		cwidth  int = int(math.Round(float64(width-woffset) * 0.02))
		theight int = min(height-hoffset, len(fls))

		mdl = model{
			keys:       defaultKeyMap(),
			readonly:   readonly,
			once:       once,
			termheight: height,
			termwidth:  width,
			mode:       mode,
			selected:   map[string]bool{},
			selectsize: 0,
			files:      fls,
		}
	)

	if workdir != "" {
		mdl.workdir = filepath.Clean(workdir)
	}

	rows := mdl.freshRows(preselected)

	var datecolumn string
	switch mdl.mode {
	case modes.Trashing:
		datecolumn = "modified"
	default:
		datecolumn = "trashed"
	}

	columns := []table.Column{
		{Title: "filename", Width: fwidth},
		{Title: "path", Width: owidth},
		{Title: datecolumn, Width: dwidth},
		{Title: "size", Width: swidth},
	}
	if !mdl.readonly {
		columns = append(columns, table.Column{Title: uncheck, Width: cwidth})
	} else {
		columns[0].Width += cwidth
	}

	mdl.table = createTable(columns, rows, theight)

	mdl.sorting = sorting.Name
	mdl.sort()

	return mdl
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.mark):
			m.toggleItem(m.table.Cursor())
		case key.Matches(msg, m.keys.doit):
			if !m.readonly && m.mode != modes.Interactive {
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

	if m.mode != modes.Listing {
		panels = append(panels, m.header())
	}
	panels = append(panels, style.Render(m.table.View()), m.footer())

	return lipgloss.JoinVertical(lipgloss.Top, panels...)
}

func (m model) showHelp() string {
	// TODO: maybe use bubbletea built in help
	var keys = []string{
		fmt.Sprintf("%s %s (%s)", darktext.Render(m.keys.sort.Help().Key), darkertext.Render(m.keys.sort.Help().Desc), m.sorting.String()),
		fmt.Sprintf("%s %s", darktext.Render(m.keys.quit.Help().Key), darkertext.Render(m.keys.quit.Help().Desc)),
	}
	if !m.readonly {
		if m.mode != modes.Interactive {
			keys = append([]string{
				fmt.Sprintf("%s %s", darktext.Render(m.keys.doit.Help().Key), darkertext.Render(m.keys.doit.Help().Desc)),
			}, keys...)
		}
		keys = append([]string{
			fmt.Sprintf("%s %s", darktext.Render(m.keys.mark.Help().Key), darkertext.Render(m.keys.mark.Help().Desc)),
		}, keys...)
	}
	return strings.Join(keys, darkesttext.Render(" • "))
}

func (m model) header() string {
	var (
		right, left string
		spacerWidth int
		keys        = []string{
			fmt.Sprintf("%s %s", darktext.Render(m.keys.rstr.Help().Key), darkertext.Render(m.keys.rstr.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.clen.Help().Key), darkertext.Render(m.keys.clen.Help().Desc)),
		}
		selectKeys = []string{
			fmt.Sprintf("%s %s", darktext.Render(m.keys.todo.Help().Key), darkertext.Render(m.keys.todo.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.nada.Help().Key), darkertext.Render(m.keys.nada.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.invr.Help().Key), darkertext.Render(m.keys.invr.Help().Desc)),
		}
		dot     = darkesttext.Render("•")
		wideDot = darkesttext.Render(" • ")
	)

	right = " " // to offset from the table border
	switch m.mode {
	case modes.Interactive:
		right += strings.Join(keys, wideDot)
	default:
		right += m.mode.String()
		if m.workdir != "" {
			right += fmt.Sprintf(" in %s", dirs.UnExpand(m.workdir, ""))
		}
	}
	right += fmt.Sprintf(" %s %s", dot, strings.Join(selectKeys, wideDot))

	left = fmt.Sprintf("%d/%d %s %s", len(m.selected), len(m.table.Rows()), dot, humanize.Bytes(uint64(m.selectsize)))

	// offset of 2 again because of table border
	spacerWidth = m.termwidth - lipgloss.Width(right) - lipgloss.Width(left) - 2
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
	if m.mode != modes.Interactive || len(m.selected) <= 0 {
		var cmd tea.Cmd
		return m, cmd
	}

	m.mode = mode
	m.onlySelected()
	m.table.SetStyles(makeUnselectedStyle())
	return m, tea.Quit
}

func (m model) selectedFiles() (outfile files.Files) {
	for _, file := range m.files {
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

func (m *model) freshRows(preselected bool) (rows []table.Row) {
	for _, file := range m.files {
		row := newRow(file, m.workdir)

		if !m.readonly {
			row = append(row, getCheck(preselected))
		}
		if preselected {
			m.selected[file.String()] = true
			m.selectsize += file.Filesize()
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
	if m.readonly {
		return false
	}

	name := m.files[index].String()
	size := m.files[index].Filesize()

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
	if m.readonly {
		return
	}

	m.selected = map[string]bool{}
	m.selectsize = 0
	for i := range m.table.Rows() {
		m.selected[m.files[i].String()] = true
		m.selectsize += m.files[i].Filesize()
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
	if m.readonly {
		return
	}

	var newrows []table.Row

	for index, row := range m.table.Rows() {
		name := m.files[index].String()
		size := m.files[index].Filesize()
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
	var rows = []table.Row{}
	for _, file := range m.files {
		r := newRow(file, m.workdir)
		if !m.readonly {
			r = append(r, getCheck(m.selected[file.String()]))
		}
		rows = append(rows, r)
	}

	m.table.SetRows(rows)
}

func Select(fls files.Files, width, height int, readonly, preselected, once bool, workdir string, mode modes.Mode) (files.Files, modes.Mode, error) {
	mdl := newModel(fls, width, height, readonly, preselected, once, workdir, mode)
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

func newRow(file files.File, workdir string) table.Row {
	var time, bar string
	time = humanize.Time(file.Date())
	if file.IsDir() {
		bar = strings.Repeat("─", 3)
	} else {
		bar = humanize.Bytes(uint64(file.Filesize()))
	}
	return table.Row{
		dirs.UnEscape(file.Name()),
		dirs.UnExpand(filepath.Dir(file.Path()), workdir),
		time,
		bar,
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
	tbl := table.DefaultKeyMap()

	// remove spacebar from default page down keybind, but keep the rest
	tbl.PageDown.SetKeys(
		slices.DeleteFunc(tbl.PageDown.Keys(), func(s string) bool {
			return s == space
		})...,
	)

	return tbl
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
