package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"essm/internal/aws"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type startSessionMsg struct{}

// table styling using lipgloss
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table     table.Model
	showTable bool
	height    int
	width     int
}

func initialModel() model {

	// instantiate the instances struct

	ec2Instances := aws.NewInstanceData()

	columns := []table.Column{
		{Title: "Instance ID", Width: 24},
		{Title: "Name", Width: 10},
		{Title: "State", Width: 16},
	}

	// slice of slices
	// Each element is a slice {} of strings
	rows := []table.Row{}

	for _, instance := range ec2Instances.Instances {
		rows = append(rows, instance.GetStringData())
	}

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
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			m.showTable = false
			return m, tea.Quit
		case "enter":
			m.showTable = false
			return m, tea.Tick(1000*time.Millisecond, func(time.Time) tea.Msg {
				return startSessionMsg{}
			})
		}
	case startSessionMsg:
		return m, tea.ExecProcess(exec.Command("aws", "ssm", "start-session", "--target", m.table.SelectedRow()[0]), nil)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 5)
		m.table.Columns()
		return m, nil
	}

	m.showTable = true // make sure table is shown if no other child process takes over the terminal

	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

func (m model) View() tea.View {

	if m.showTable {
		return tea.NewView(baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n")
	}

	return tea.NewView("Connecting...")

}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
