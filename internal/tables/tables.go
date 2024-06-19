package tables

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"git.burning.moe/celediel/gt/internal/dirs"
	"git.burning.moe/celediel/gt/internal/files"
	"git.burning.moe/celediel/gt/internal/trash"
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
	hoffset int    = 5

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
	selected   []int
	readonly   bool
	termheight int
}

// TODO: reconcile trash.Info and files.File into an interface so I can shorten this up

func newInfosModel(is trash.Infos, width, height int, readonly, preselected bool) model {
	var (
		fwidth  int = int(math.Round(float64(width-woffset) * 0.4))
		owidth  int = int(math.Round(float64(width-woffset) * 0.2))
		dwidth  int = int(math.Round(float64(width-woffset) * 0.25))
		swidth  int = int(math.Round(float64(width-woffset) * 0.12))
		cwidth  int = int(math.Round(float64(width-woffset) * 0.03))
		theight int = min(height-hoffset, len(is))

		m = model{
			keys:       defaultKeyMap(),
			readonly:   readonly,
			termheight: height,
		}
	)
	slices.SortStableFunc(is, trash.SortByTrashedReverse)

	rows := []table.Row{}
	for j, i := range is {
		var t, b string
		t = humanize.Time(i.Trashed())
		if i.IsDir() {
			b = strings.Repeat("─", 3)
		} else {
			b = humanize.Bytes(uint64(i.Filesize()))
		}
		r := table.Row{
			i.Name(),
			dirs.UnExpand(filepath.Dir(i.OGPath())),
			t,
			b,
		}

		if !m.readonly {
			r = append(r, getCheck(preselected))
		}
		if preselected {
			m.selected = append(m.selected, j)
		}
		rows = append(rows, r)
	}

	columns := []table.Column{
		{Title: "filename", Width: fwidth},
		{Title: "original path", Width: owidth},
		{Title: "deleted", Width: dwidth},
		{Title: "size", Width: swidth},
	}
	if !m.readonly {
		columns = append(columns, table.Column{Title: uncheck, Width: cwidth})
	} else {
		columns[0].Width += cwidth
	}

	m.table = createTable(columns, rows, theight, m.readonlyOnePage())

	return m
}

func newFilesModel(fs files.Files, width, height int, readonly, preselected bool) model {
	var (
		fwidth  int = int(math.Round(float64(width-woffset) * 0.4))
		owidth  int = int(math.Round(float64(width-woffset) * 0.2))
		dwidth  int = int(math.Round(float64(width-woffset) * 0.25))
		swidth  int = int(math.Round(float64(width-woffset) * 0.12))
		cwidth  int = int(math.Round(float64(width-woffset) * 0.03))
		theight int = min(height-hoffset, len(fs))

		m = model{
			keys:     defaultKeyMap(),
			readonly: readonly,
		}
	)

	slices.SortStableFunc(fs, files.SortByModifiedReverse)

	rows := []table.Row{}
	for j, f := range fs {
		var t, b string
		t = humanize.Time(f.Modified())
		if f.IsDir() {
			b = strings.Repeat("─", 3)
		} else {
			b = humanize.Bytes(uint64(f.Filesize()))
		}
		r := table.Row{
			f.Name(),
			dirs.UnExpand(f.Path()),
			t,
			b,
		}

		if !m.readonly {
			r = append(r, getCheck(preselected))
		}
		if preselected {
			m.selected = append(m.selected, j)
		}
		rows = append(rows, r)
	}

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

	return m
}

type keyMap struct {
	mark key.Binding
	doit key.Binding
	quit key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		mark: key.NewBinding(
			key.WithKeys(space),
			key.WithHelp("space", "toggle selected"),
		),
		doit: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "confirm"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/^c", "quit"),
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
			m.select_item(m.table.SelectedRow(), m.table.Cursor())
		case key.Matches(msg, m.keys.doit):
			if !m.readonly {
				return m, tea.Quit
			}
		case key.Matches(msg, m.keys.quit):
			m.selected = []int{}
			return m, tea.Quit
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

	out = lipgloss.JoinVertical(lipgloss.Top, panels...)
	return out + n
}

func (m model) readonlyOnePage() bool {
	return m.readonly && m.termheight > m.table.Height()
}

func (m model) showHelp() string {
	// TODO: maybe use bubbletea built in help
	var keys []string = []string{
		fmt.Sprintf("%s %s", darktext.Render(m.keys.quit.Help().Key), darkertext.Render(m.keys.quit.Help().Desc)),
	}
	if !m.readonly {
		keys = append([]string{
			fmt.Sprintf("%s %s", darktext.Render(m.keys.mark.Help().Key), darkertext.Render(m.keys.mark.Help().Desc)),
			fmt.Sprintf("%s %s", darktext.Render(m.keys.doit.Help().Key), darkertext.Render(m.keys.doit.Help().Desc)),
		}, keys...)
	}
	return strings.Join(keys, darkesttext.Render(" • "))
}

func (m model) footer() string {
	return regulartext.Render(m.showHelp())
}

// select_item selects an item, and returns its current selected state
func (m *model) select_item(row table.Row, index int) (selected bool) {
	if m.readonly {
		return false
	}

	// select the thing
	if slices.Contains(m.selected, index) {
		// already selected
		m.selected = slices.DeleteFunc(m.selected, func(other int) bool { return index == other })
		selected = false
	} else {
		// not selected
		m.selected = append(m.selected, index)
		selected = true
	}

	// update the rows with the state
	rows := m.table.Rows()
	rows[index] = table.Row{
		row[0],
		row[1],
		row[2],
		row[3],
		getCheck(selected),
	}

	m.table.SetRows(rows)

	return
}

func InfoTable(is trash.Infos, width, height int, readonly, preselected bool) ([]int, error) {
	if endmodel, err := tea.NewProgram(newInfosModel(is, width, height, readonly, preselected)).Run(); err != nil {
		return []int{}, err
	} else {
		m, ok := endmodel.(model)
		if ok {
			return m.selected, nil
		} else {
			return []int{}, fmt.Errorf("model isn't the right type??")
		}
	}
}

func FilesTable(fs files.Files, width, height int, readonly, preselected bool) ([]int, error) {
	if endmodel, err := tea.NewProgram(newFilesModel(fs, width, height, readonly, preselected)).Run(); err != nil {
		return []int{}, err
	} else {
		m, ok := endmodel.(model)
		if ok {
			return m.selected, nil
		} else {
			return []int{}, fmt.Errorf("model isn't the right type??")
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
		style := makeStyle()
		style.Selected = style.Selected.
			Foreground(lipgloss.NoColor{}).
			Background(lipgloss.NoColor{}).
			Bold(false)

		t.SetStyles(style)
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
