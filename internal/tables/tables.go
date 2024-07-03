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
	table       table.Model
	keys        keyMap
	selected    map[int]bool
	readonly    bool
	preselected bool
	termheight  int
	mode        modes.Mode
	sorting     sorting.Sorting
	workdir     string
	files       files.Files
}

func newModel(fs []files.File, width, height int, readonly, preselected bool, workdir string, mode modes.Mode) model {
	var (
		fwidth  int = int(math.Round(float64(width-woffset) * 0.4))
		owidth  int = int(math.Round(float64(width-woffset) * 0.2))
		dwidth  int = int(math.Round(float64(width-woffset) * 0.25))
		swidth  int = int(math.Round(float64(width-woffset) * 0.12))
		cwidth  int = int(math.Round(float64(width-woffset) * 0.03))
		theight int = min(height-hoffset, len(fs))

		m = model{
			keys:        defaultKeyMap(),
			readonly:    readonly,
			preselected: preselected,
			termheight:  height,
			mode:        mode,
			selected:    map[int]bool{},
			workdir:     workdir,
			files:       fs,
		}
	)

	rows := m.makeRows()

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

	m.table = createTable(columns, rows, theight, m.readonlyOnePage())

	m.sorting = sorting.Size
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
	if m.readonlyOnePage() {
		return tea.Quit
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

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
			if m.mode == modes.Interactive {
				m.mode = modes.Cleaning
				return m.quit(false)
			}
		case key.Matches(msg, m.keys.rstr):
			if m.mode == modes.Interactive {
				m.mode = modes.Restoring
				return m.quit(false)
			}
		case key.Matches(msg, m.keys.sort):
			// if !m.readonly {
			m.sorting = m.sorting.Next()
			m.sort()
			// }
		case key.Matches(msg, m.keys.rort):
			// if !m.readonly {
			m.sorting = m.sorting.Prev()
			m.sort()
			// }
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

	if m.readonlyOnePage() {
		n = "\n"
	} else {
		panels = append(panels, m.footer())
	}

	if m.mode != modes.Listing {
		panels = append([]string{m.header()}, panels...)
	}

	out = lipgloss.JoinVertical(lipgloss.Top, panels...)
	return out + n
}

func (m model) readonlyOnePage() bool {
	return m.readonly && m.termheight > m.table.Height()
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

func (m *model) makeRows() (rows []table.Row) {
	for j, f := range m.files {
		var t, b string
		t = humanize.Time(f.Date())
		if f.IsDir() {
			b = strings.Repeat("─", 3)
		} else {
			b = humanize.Bytes(uint64(f.Filesize()))
		}
		r := table.Row{
			dirs.UnEscape(f.Name()),
			dirs.UnExpand(filepath.Dir(f.Path()), m.workdir),
			t,
			b,
		}

		if !m.readonly {
			r = append(r, getCheck(m.preselected))
		}
		if m.preselected {
			m.selected[j] = true
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

// toggleItem toggles an item's selected state, and returns the state
func (m *model) toggleItem(index int) (selected bool) {
	if m.readonly {
		return false
	}

	// select the thing
	if v, ok := m.selected[index]; v && ok {
		// already selected
		delete(m.selected, index)
		selected = false
	} else {
		// not selected
		m.selected[index] = true
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

	m.selected = map[int]bool{}
	for i := range len(m.table.Rows()) {
		m.selected[i] = true
	}
	m.updateRows(true)
}

func (m *model) unselectAll() {
	if m.readonly {
		return
	}

	m.selected = map[int]bool{}
	m.updateRows(false)
}

func (m *model) invertSelection() {
	var newrows []table.Row

	for index, row := range m.table.Rows() {
		if v, ok := m.selected[index]; v && ok {
			delete(m.selected, index)
			newrows = append(newrows, table.Row{
				row[0],
				row[1],
				row[2],
				row[3],
				getCheck(false),
			})
		} else {
			m.selected[index] = true
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
	m.table.SetRows(m.makeRows())
}

func Show(fs []files.File, width, height int, readonly, preselected bool, workdir string, mode modes.Mode) ([]int, modes.Mode, error) {
	if endmodel, err := tea.NewProgram(newModel(fs, width, height, readonly, preselected, workdir, mode)).Run(); err != nil {
		return []int{}, 0, err
	} else {
		m, ok := endmodel.(model)
		if ok {
			selected := make([]int, 0, len(m.selected))
			for k := range m.selected {
				selected = append(selected, k)
			}

			return selected, m.mode, nil
		} else {
			return []int{}, 0, fmt.Errorf("model isn't the right type??")
		}
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

func createTable(columns []table.Column, rows []table.Row, height int, readonlyonepage bool) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	t.KeyMap = fixTableKeymap()
	if readonlyonepage {
		t.SetStyles(makeUnselectedStyle())
	} else {
		t.SetStyles(makeStyle())
	}
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
