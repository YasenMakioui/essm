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
}

func fetchInstances() tea.Msg {
	return instancesLoadedMsg{instances: aws.NewInstanceData()}
}

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

	// slice of slices
	// Each element is a slice {} of strings
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
	return fetchInstances
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.searchTarget {
				m.searchTarget = false
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
			m.showTable = false
			return m, tea.Tick(1000*time.Millisecond, func(time.Time) tea.Msg {
				return startSessionMsg{}
			})
		case "/":
			if m.searchTarget {
				// search is already toggled. Add the trailing slash to the string
				m.searchText = m.searchText + "/"
			}
			m.searchTarget = true
			return m, cmd
		default: // handle text typing. This should be another model after finishing it. Extract to its own function
			if m.searchTarget {
				// pass in the pressed key and the current text
				searchVal := parseSearchValue(msg.String(), m.searchText)

				// assign parsed text
				m.searchText = searchVal

				// update table highlighting
				// TODO: handle the table view here removing rows that do not match with the search text
				return m, cmd
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

	m.showTable = true // make sure table is shown if no other child process takes over the terminal

	m.table, cmd = m.table.Update(msg) // keys for navigation work here

	return m, cmd
}

func (m model) View() tea.View {

	if m.showTable {
		if !m.searchTarget {
			return tea.NewView(baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n")
		}
		// search could be another model with its view and functionalities. After making it work, extract it.
		return tea.NewView(baseStyle.Render(m.table.View()) + "\n " + "Search: " + m.searchText + "\n " + m.table.HelpView() + "\n")
	}

	return tea.NewView("Connecting...")

}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
