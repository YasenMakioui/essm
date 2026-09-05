package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/YasenMakioui/essm/internal/aws"
	"github.com/YasenMakioui/essm/internal/models"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type errMsg struct{ err error }

type startSessionMsg struct{}

type appMode int

const (
	modeTable appMode = iota
	modeConnecting
)

// table styling using lipgloss
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("240"))

var (
	ssmOnlineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green
	ssmOfflineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // dim grey
)

const refreshInterval = 30 * time.Second

const (
	tableBorderWidth = 2 // 1 char each side, from baseStyle's NormalBorder
	tableCellPadding = 2 // 1 char each side per column, from table.DefaultStyles()
	tableNumColumns  = 4
)

type model struct {
	table       models.Table
	searchInput textinput.Model // replaces searchText string
	searching   bool            // replaces searchTarget (rename for clarity)
	info        models.InfoPanel
	infoID      string
	mode        appMode
	showInfo    bool
	height      int
	width       int
	loading     bool
	updatedAt   time.Time
	err         error
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	instances, err := aws.FetchInstances(ctx)
	if err != nil {
		return errMsg{err}
	}
	return rowsMsg{rows: instancesToRows(instances), fetchedAt: time.Now()}
}

type detailMsg struct{ detail aws.InstanceDetail }
type detailErrMsg struct {
	id  string
	err error
}
type sessionEndedMsg struct{ err error }

func fetchDetail(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		d, err := aws.FetchInstanceDetail(ctx, id)
		if err != nil {
			return detailErrMsg{id: id, err: err}
		}
		return detailMsg{d}
	}
}

// instancesToRows maps domain data to table rows — the UI/AWS boundary.
func instancesToRows(instances []aws.Instance) []table.Row {
	rows := make([]table.Row, 0, len(instances))
	for _, in := range instances {
		rows = append(rows, table.Row{in.ID, in.Name, in.State, ssmGlyph(in.SSM)})
	}
	return rows
}

func ssmGlyph(s aws.SSMStatus) string {
	switch s {
	case aws.SSMOnline:
		return ssmOnlineStyle.Render("● online")
	case aws.SSMOffline:
		return ssmOfflineStyle.Render("○ offline")
	default:
		return "-"
	}
}

func (m *model) applyLayout() {
	infoW := 0
	if m.showInfo {
		infoW = m.width / 3
	}
	m.table.SetSize(m.width-infoW, m.height-5)
	m.info.SetSize(infoW-2, m.height-5)
}

func initialModel() model {
	ti := textinput.New()
	ti.Prompt = "Search: "
	ti.Placeholder = "id or name"

	return model{
		table:       models.NewTable(),
		searchInput: ti,
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
		// global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.searching {
				m.searching = false
				m.searchInput.Blur()
				m.searchInput.Reset()
				m.table.ClearFilter()
				return m, nil
			}

			if m.showInfo {
				m.showInfo = false
				m.applyLayout()
				return m, nil
			}
			return m, nil

		case "enter":
			if !m.table.HasSelection() {
				return m, nil
			}
			m.mode = modeConnecting
			return m, func() tea.Msg { return startSessionMsg{} }
		}

		// in search mode: arrows drive the table, everything else feeds the input
		if m.searching {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown":
				cmd = m.table.Update(msg)
				return m, cmd
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.table.SetFilter(m.searchInput.Value())
				return m, cmd
			}
		}

		// not searching: table-mode keys
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "/":
			m.searching = true
			return m, m.searchInput.Focus()
		case "r":
			if !m.loading {
				m.loading = true
				return m, fetchRows
			}
			return m, nil
		case "i":
			id := m.table.SelectedID()
			if id == "" {
				return m, nil
			}
			// i on the instance already shown closes the panel; i on a different one updates it
			if m.showInfo && m.infoID == id {
				m.showInfo = false
				m.applyLayout()
				return m, nil
			}
			m.showInfo = true
			m.infoID = id
			m.info.SetLoading()
			m.applyLayout()
			return m, fetchDetail(id)
		}
	case rowsMsg:
		m.updatedAt = msg.fetchedAt
		m.table.SetRows(msg.rows)
		m.loading = false
		m.err = nil
		return m, nil
	case tickMsg: // refresh rows
		if m.loading {
			return m, nil
		}
		m.loading = true
		return m, tea.Batch(tickCmd(), fetchRows)
	case startSessionMsg:
		return m, tea.ExecProcess(
			exec.Command("aws", "ssm", "start-session", "--target", m.table.SelectedID()),
			func(err error) tea.Msg { return sessionEndedMsg{err} },
		)
	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
	case detailMsg:
		if msg.detail.ID == m.infoID {
			m.info.SetDetail(msg.detail)
		}
		return m, nil
	case detailErrMsg:
		if msg.id == m.infoID {
			m.info.SetError(msg.err)
		}
		return m, nil
	case sessionEndedMsg:
		m.mode = modeTable
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyLayout()
		return m, nil
	}

	cmd = m.table.Update(msg) // for any other unhandled case

	return m, cmd
}

func (m model) View() tea.View {
	if m.mode == modeConnecting {
		v := tea.NewView("Connecting...")
		v.AltScreen = true
		return v
	}

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
			"last update: %s  •  %s  •  every %s\n↑/↓ move · / search · i info · r refresh · q quit",
			stamp, status, refreshInterval,
		))

	if m.err != nil {
		footer = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render("error: "+m.err.Error()) + "\n" + footer
	}

	body := m.table.View()
	if m.showInfo {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, m.info.View())
	}
	body += "\n"
	if m.searching {
		body += m.searchInput.View() + "\n"
	}
	body += footer + "\n"
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
