package tables

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"git.burning.moe/celediel/gt/internal/dirs"
	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/tables/modes"
	"git.burning.moe/celediel/gt/internal/tables/sorting"

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
	readonly   bool
	once       bool
	termheight int
	mode       modes.Mode
	sorting    sorting.Sorting
	workdir    string
	files      files.Files
}

func newModel(fs []files.File, width, height int, readonly, preselected, once bool, workdir string, mode modes.Mode) model {
	var (
		// TODO: figure this out dynamically based on longest of each
		fwidth  int = int(math.Round(float64(width-woffset) * 0.46))
		owidth  int = int(math.Round(float64(width-woffset) * 0.25))
		dwidth  int = int(math.Round(float64(width-woffset) * 0.15))
		swidth  int = int(math.Round(float64(width-woffset) * 0.12))
		cwidth  int = int(math.Round(float64(width-woffset) * 0.02))
		theight int = min(height-hoffset, len(fs))

		m = model{
			keys:       defaultKeyMap(),
			readonly:   readonly,
			once:       once,
			termheight: height,
			mode:       mode,
			selected:   map[string]bool{},
			files:      fs,
		}
	)

	if workdir != "" {
		m.workdir = filepath.Clean(workdir)
	}

	rows := m.freshRows(preselected)

	columns := []table.Column{
		{Title: "filename", Width: fwidth},
		{Title: "path", Width: owidth},
		{Title: "modified", Width: dwidth},
		{Title: "size", Width: swidth},
	}
	if !m.readonly {
		columns = append(columns, table.Column{Title: uncheck, Width: cwidth})
	} else {
		columns[0].Width += cwidth
	}

	m.table = createTable(columns, rows, theight)

	m.sorting = sorting.Name
	m.sort()

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
			key.WithHelp("a", "select all"),
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
			key.WithHelp("s", "sort"),
		),
		rort: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "change sort (reverse)"),
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
			if m.mode == modes.Interactive && len(m.selected) > 0 {
				m.mode = modes.Cleaning
				return m.quit(false)
			}
		case key.Matches(msg, m.keys.rstr):
			if m.mode == modes.Interactive && len(m.selected) > 0 {
				m.mode = modes.Restoring
				return m.quit(false)
			}
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

func (m model) View() (out string) {
	var (
		n      string
		panels []string = []string{
			style.Render(m.table.View()),
		}
	)

	panels = append(panels, m.footer())

	if m.mode != modes.Listing {
		panels = append([]string{m.header()}, panels...)
	}

	out = lipgloss.JoinVertical(lipgloss.Top, panels...)
	return out + n
}

func (m model) showHelp() string {
	// TODO: maybe use bubbletea built in help
	var keys []string = []string{
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
		mode string
		keys = []string{
			fmt.Sprintf("%s %s", darktext.Render(m.keys.rstr.Help().Key), darkertext.Render(m.keys.rstr.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.clen.Help().Key), darkertext.Render(m.keys.clen.Help().Desc)),
		}
		select_keys = []string{
			fmt.Sprintf("%s %s", darktext.Render(m.keys.todo.Help().Key), darkertext.Render(m.keys.todo.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.nada.Help().Key), darkertext.Render(m.keys.nada.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.invr.Help().Key), darkertext.Render(m.keys.invr.Help().Desc)),
		}
		dot      = darkesttext.Render("•")
		wide_dot = darkesttext.Render(" • ")
	)

	switch m.mode {
	case modes.Interactive:
		mode = strings.Join(keys, wide_dot)
	default:
		mode = m.mode.String()
		if m.workdir != "" {
			mode += fmt.Sprintf(" in %s ", dirs.UnExpand(m.workdir, ""))
		}
	}
	mode += fmt.Sprintf(" %s %s", dot, strings.Join(select_keys, wide_dot))

	return fmt.Sprintf(" %s %s %d/%d", mode, dot, len(m.selected), len(m.table.Rows()))
}

func (m model) footer() string {
	return regulartext.Render(m.showHelp())
}

func (m model) quit(unselect_all bool) (model, tea.Cmd) {
	if unselect_all {
		m.unselectAll()
	} else {
		m.onlySelected()
	}
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

func newRow(file files.File, workdir string) table.Row {
	var t, b string
	t = humanize.Time(file.Date())
	if file.IsDir() {
		b = strings.Repeat("─", 3)
	} else {
		b = humanize.Bytes(uint64(file.Filesize()))
	}
	return table.Row{
		dirs.UnEscape(file.Name()),
		dirs.UnExpand(filepath.Dir(file.Path()), workdir),
		t,
		b,
	}
}

func (m *model) freshRows(preselected bool) (rows []table.Row) {
	for _, f := range m.files {
		r := newRow(f, m.workdir)

		if !m.readonly {
			r = append(r, getCheck(preselected))
		}
		if preselected {
			m.selected[f.String()] = true
		}
		rows = append(rows, r)
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

// updateRow updates row of `index` with `row`
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
	var newrows []table.Row

	for _, row := range m.table.Rows() {
		r := table.Row{
			row[0],
			row[1],
			row[2],
			row[3],
			getCheck(selected),
		}
		newrows = append(newrows, r)
	}
	m.table.SetRows(newrows)
}

func (m *model) toggleItem(index int) (selected bool) {
	if m.readonly {
		return false
	}

	name := m.files[index].String()

	// select the thing
	if v, ok := m.selected[name]; v && ok {
		// already selected
		delete(m.selected, name)
		selected = false
	} else {
		// not selected
		m.selected[name] = true
		selected = true
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
	for i := range m.table.Rows() {
		m.selected[m.files[i].String()] = true
	}
	m.updateRows(true)
}

func (m *model) unselectAll() {
	if m.readonly {
		return
	}

	m.selected = map[string]bool{}
	m.updateRows(false)
}

func (m *model) invertSelection() {
	if m.readonly {
		return
	}

	var newrows []table.Row

	for index, row := range m.table.Rows() {
		name := m.files[index].String()
		if v, ok := m.selected[name]; v && ok {
			delete(m.selected, name)
			newrows = append(newrows, table.Row{
				row[0],
				row[1],
				row[2],
				row[3],
				getCheck(false),
			})
		} else {
			m.selected[name] = true
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
	var rows []table.Row
	for _, file := range m.files {
		r := newRow(file, m.workdir)
		if !m.readonly {
			r = append(r, getCheck(m.selected[file.String()]))
		}
		rows = append(rows, r)
	}

	m.table.SetRows(rows)
}

func Select(fs files.Files, width, height int, readonly, preselected, once bool, workdir string, mode modes.Mode) (files.Files, modes.Mode, error) {
	mdl := newModel(fs, width, height, readonly, preselected, once, workdir, mode)
	endmodel, err := tea.NewProgram(mdl).Run()
	if err != nil {
		return fs, 0, err
	}
	m, ok := endmodel.(model)
	if !ok {
		return fs, 0, fmt.Errorf("model isn't the right type?? what has happened")
	}
	return m.selectedFiles(), m.mode, nil
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
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	t.KeyMap = fixTableKeymap()
	t.SetStyles(makeStyle())
	return t
}

func fixTableKeymap() table.KeyMap {
	t := table.DefaultKeyMap()

	// remove spacebar from default page down keybind, but keep the rest
	t.PageDown.SetKeys(
		slices.DeleteFunc(t.PageDown.Keys(), func(s string) bool {
			return s == space
		})...,
	)

	return t
}

func makeStyle() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(black)).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(white)).
		Background(lipgloss.Color(hoveritembg)).
		Bold(false)

	return s
}

func makeUnselectedStyle() table.Styles {
	style := makeStyle()
	style.Selected = style.Selected.
		Foreground(lipgloss.NoColor{}).
		Background(lipgloss.NoColor{}).
		Bold(false)
	return style
}
