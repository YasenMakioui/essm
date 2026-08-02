package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"essm/internal/aws"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type startSessionMsg struct{}

type instancesLoadedMsg struct {
	instances aws.InstanceData
}

// table styling using lipgloss
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("240"))

const refreshInterval = 30 * time.Second

const (
	tableBorderWidth = 2 // 1 char each side, from baseStyle's NormalBorder
	tableCellPadding = 2 // 1 char each side per column, from table.DefaultStyles()
	tableNumColumns  = 3
)

type model struct {
	table        table.Model
	showTable    bool
	height       int
	width        int
	searchTarget bool
	searchText   string
	loading      bool
	updatedAt    time.Time
	currentRows  []table.Row
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Every(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type rowsMsg struct {
	rows      []table.Row
	fetchedAt time.Time
}

// Generate the table rows
func fetchRows() tea.Msg {
	i := aws.NewInstanceData()
	rows := []table.Row{}

	for _, instance := range i.Instances {
		rows = append(rows, instance.GetStringData())
	}

	return rowsMsg{rows: rows, fetchedAt: time.Now()}
}

func filterRows(data []table.Row, filter string) []table.Row {
	var filteredRows []table.Row

	// if filter string is empty, return current rows
	if filter == "" {
		return data
	}

	for _, r := range data {
		if strings.Contains(r[1], filter) {
			filteredRows = append(filteredRows, r)
		}
	}

	return filteredRows
}

// imorove function, this is a PoC right now
func parseSearchValue(s, currMsg string) string {
	// gets a string message from a key pressed.
	// If the message is something like backspace, tab and so on, handle it.

	switch s {
	case "space":
		return currMsg + " "
	case "backspace":
		// avoid issues on empty strings
		if currMsg == "" {
			return currMsg
		}

		// subtract last char from the current mesasge

		return currMsg[:len(currMsg)-1]
	case "tab":
		return currMsg // don't handle tabs, this is an instance finder, not a text editor.
	case "up":
		return currMsg
	case "right":
		return currMsg
	case "down":
		return currMsg
	case "left":
		return currMsg
	default:
		if strings.HasPrefix(s, "ctrl+") {
			return currMsg
		}

		if s == "delete" {
			return currMsg
		}

		return currMsg + s
		// f keys, ctrl + combinations and so on
	}
}

func initialModel() model {

	columns := []table.Column{
		{Title: "Instance ID", Width: 24},
		{Title: "Name", Width: 10},
		{Title: "State", Width: 16},
	}

	rows := []table.Row{}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
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

	return model{
		table:     t,
		showTable: true,
	}
}

// No startup I/O, so no initial command.
func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), fetchRows)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.searchTarget {
				m.searchTarget = false

				// recover all tables from struct val
				m.table.SetRows(m.currentRows)

				// unser curr rows in case another search is issued
				m.currentRows = nil
				m.searchText = ""
				return m, cmd
			}

			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			m.showTable = false
			if m.searchTarget {
				if msg.String() != "ctrl+c" {
					m.searchText = m.searchText + msg.String()
				} else {
					return m, tea.Quit
				}
			} else {
				return m, tea.Quit
			}
		case "enter":
			// if there's no rows ignore, otherwise a panic will raise
			if len(m.table.SelectedRow()) == 0 {
				return m, nil
			}

			m.showTable = false
			return m, tea.Tick(1000*time.Millisecond, func(time.Time) tea.Msg {
				return startSessionMsg{}
			})
		case "/":
			// before toggle search target, save current rows in mem
			// if its nil, then just asign the current rows. If it isn't that means that mem is being used
			if m.currentRows == nil {
				m.currentRows = m.table.Rows()
			}
			if m.searchTarget {
				// search is already toggled. Add the trailing slash to the string
				m.searchText = m.searchText + "/"
				return m, cmd
			}
			m.searchTarget = true
			return m, cmd
		case "r": // Refresh the table. Fetch
			if m.searchTarget {
				m.searchText = m.searchText + "r"
				return m, cmd
			}
			if !m.loading {
				m.loading = true
				return m, fetchRows
			}
			return m, nil
		default: // handle text typing. This should be another model after finishing it. Extract to its own function
			if m.searchTarget {
				// keys that do not need handling like arrow keys need to be passed to the table update below.
				if !(msg.String() == "up" || msg.String() == "down") {
					// pass in the pressed key and the current text
					searchVal := parseSearchValue(msg.String(), m.searchText)

					// assign parsed text
					m.searchText = searchVal

					// always filter against saved rows pre-search
					filteredRows := filterRows(m.currentRows, m.searchText)
					m.table.SetRows(filteredRows)

					// update table highlighting
					// TODO: handle the table view here removing rows that do not match with the search text
					return m, cmd
				}
			}
		}
	case instancesLoadedMsg:
		rows := []table.Row{}
		for _, instance := range msg.instances.Instances {
			rows = append(rows, instance.GetStringData())
		}
		m.table.SetRows(rows)
		return m, nil
	case startSessionMsg:
		return m, tea.ExecProcess(exec.Command("aws", "ssm", "start-session", "--target", m.table.SelectedRow()[0]), nil)
	case tickMsg:
		if m.loading {
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(tickCmd(), fetchRows)
	case rowsMsg:
		m.updatedAt = msg.fetchedAt
		// do not set rows if search mode is toggled
		if !m.searchTarget {
			m.table.SetRows(msg.rows)
		}
		m.loading = false
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.table.SetWidth(msg.Width - tableBorderWidth)
		m.table.SetHeight(msg.Height - 5)

		available := msg.Width - tableBorderWidth - tableCellPadding*tableNumColumns
		m.table.SetColumns([]table.Column{
			{Title: "Instance ID", Width: available * 40 / 100},
			{Title: "Name", Width: available * 30 / 100},
			{Title: "State", Width: available * 30 / 100},
		})

		return m, nil
	}

	// Default handling
	// If there is no search mode, session started or other action, we show the table
	// and pass the message to sub-models

	m.showTable = true // make sure table is shown if no other child process takes over the terminal

	m.table, cmd = m.table.Update(msg) // keys for navigation work here

	return m, cmd
}

func (m model) View() tea.View {

	status := "up to date"

	if m.loading {
		status = "refreshing..."
	}

	stamp := "-"

	if !m.updatedAt.IsZero() {
		stamp = m.updatedAt.Format("15:04:05")
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf(
			"last update: %s  •  %s  •  every %s\n↑/↓ move · r refresh · esc focus · q quit",
			stamp, status, refreshInterval,
		))

	if m.showTable {
		if !m.searchTarget {
			return tea.NewView(baseStyle.Render(m.table.View()) + "\n" + footer + "\n")
		}
		// search could be another model with its view and functionalities. After making it work, extract it.
		return tea.NewView(baseStyle.Render(m.table.View()) + "\n " + "Search: " + m.searchText + "\n" + footer + "\n")
	}

	return tea.NewView("Connecting...")

}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
