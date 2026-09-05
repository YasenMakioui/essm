package models

import (
	"fmt"
	"sort"
	"strings"

	"github.com/YasenMakioui/essm/internal/aws"

	"charm.land/lipgloss/v2"
)

// InfoPanel is a view-only component: the parent drives its state, it renders.
// It can grow an Update method later if it needs to own key events.
type InfoPanel struct {
	detail  aws.InstanceDetail
	hasData bool
	loading bool
	err     error
	width   int
	height  int
}

func (p *InfoPanel) SetLoading() { p.loading = true; p.err = nil }
func (p *InfoPanel) SetDetail(d aws.InstanceDetail) {
	p.detail = d
	p.hasData = true
	p.loading = false
	p.err = nil
}
func (p *InfoPanel) SetError(err error) { p.err = err; p.loading = false }
func (p *InfoPanel) SetSize(w, h int)   { p.width, p.height = w, h }

func (p InfoPanel) View() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(p.width).
		Height(p.height)

	label := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	switch {
	case p.loading:
		return box.Render("loading instance info…")
	case p.err != nil:
		return box.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).
			Render("error: " + p.err.Error()))
	case !p.hasData:
		return box.Render("press i for details")
	}

	d := p.detail
	var b strings.Builder
	row := func(k, v string) {
		if v == "" {
			v = "-"
		}
		fmt.Fprintf(&b, "%s %s\n", label.Render(fmt.Sprintf("%-11s", k)), v)
	}
	row("id", d.ID)
	row("type", d.Type)
	row("ami", d.AMI)
	row("private ip", d.PrivateIP)
	row("public ip", d.PublicIP)
	row("subnet", d.SubnetID)
	row("vpc", d.VpcID)
	row("iam", d.IAMProfile)

	b.WriteString("\n" + label.Render("volumes") + "\n")
	if len(d.Volumes) == 0 {
		b.WriteString("  -\n")
	}
	for _, v := range d.Volumes {
		fmt.Fprintf(&b, "  %s  %s\n", v.Device, v.ID)
	}
	b.WriteString("\n" + label.Render("security groups") + "\n")
	if len(d.SecurityGroups) == 0 {
		b.WriteString("  -\n")
	}
	for _, sg := range d.SecurityGroups {
		fmt.Fprintf(&b, "  %s  %s\n", sg.ID, sg.Name)
	}
	b.WriteString("\n" + label.Render("tags") + "\n")
	if len(d.Tags) == 0 {
		b.WriteString("  -\n")
	} else {
		keys := make([]string, 0, len(d.Tags))
		for k := range d.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys) // maps iterate randomly; sort so the panel doesn't jitter
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s = %s\n", k, d.Tags[k])
		}
	}

	return box.Render(b.String())
}
