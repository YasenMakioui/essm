package models

import (
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	borderWidth = 2 // table border: 1 char each side
	cellPadding = 2 // table.DefaultStyles: 1 char each side per column
	numColumns  = 4
)

var tableBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("240"))

// Table wraps a bubbles table plus its canonical (unfiltered) rows and the
// active search query, so the parent never manages row/filter bookkeeping.
type Table struct {
	model   table.Model
	allRows []table.Row
	query   string
}

func NewTable() Table {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Instance ID", Width: 24},
			{Title: "Name", Width: 10},
			{Title: "State", Width: 16},
			{Title: "SSM", Width: 10},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(7),
		table.WithWidth(50),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return Table{model: t}
}

// SetSize sets the table's total footprint (border included) and recomputes columns.
func (t *Table) SetSize(w, h int) {
	inner := w - borderWidth
	if inner < 0 {
		inner = 0
	}
	t.model.SetWidth(inner)
	t.model.SetHeight(h)

	available := inner - cellPadding*numColumns
	if available < numColumns {
		available = numColumns
	}
	t.model.SetColumns([]table.Column{
		{Title: "Instance ID", Width: available * 34 / 100},
		{Title: "Name", Width: available * 26 / 100},
		{Title: "State", Width: available * 20 / 100},
		{Title: "SSM", Width: available * 20 / 100},
	})
}

// SetRows replaces the canonical rows and re-applies the active filter.
func (t *Table) SetRows(rows []table.Row) {
	t.allRows = rows
	t.apply()
}

// SetFilter updates the search query and re-filters the visible rows.
func (t *Table) SetFilter(q string) {
	t.query = q
	t.apply()
}

// ClearFilter drops the query, restoring all rows.
func (t *Table) ClearFilter() {
	t.query = ""
	t.apply()
}

func (t *Table) apply() {
	t.model.SetRows(filterRows(t.allRows, t.query))
}

// SelectedID returns the instance ID of the highlighted row, or "" if none.
func (t Table) SelectedID() string {
	r := t.model.SelectedRow()
	if len(r) == 0 {
		return ""
	}
	return r[0]
}

func (t Table) HasSelection() bool {
	return len(t.model.SelectedRow()) > 0
}

func (t *Table) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return cmd
}

func (t Table) View() string {
	return tableBorder.Render(t.model.View())
}

func filterRows(rows []table.Row, q string) []table.Row {
	if q == "" {
		return rows
	}
	q = strings.ToLower(q)
	var out []table.Row
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r[0]), q) || // ID
			strings.Contains(strings.ToLower(r[1]), q) { // Name
			out = append(out, r)
		}
	}
	return out
}
