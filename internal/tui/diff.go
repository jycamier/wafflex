package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jycamier/wafflex/internal/diff"
	"github.com/jycamier/wafflex/internal/models"
)

type diffItem struct {
	entry diff.DiffEntry
}

func (i diffItem) Title() string {
	var prefix string
	var req *models.HTTPRequest

	switch i.entry.Type {
	case diff.DiffTypeAdded:
		prefix = "[+]"
		req = &i.entry.Result2.Request
	case diff.DiffTypeRemoved:
		prefix = "[-]"
		req = &i.entry.Result1.Request
	case diff.DiffTypeModified:
		prefix = "[~]"
		req = &i.entry.Result1.Request
	}

	return fmt.Sprintf("%s %s %s", prefix, req.Method, req.URL)
}

func (i diffItem) Description() string {
	switch i.entry.Type {
	case diff.DiffTypeAdded:
		return "New blocked request"
	case diff.DiffTypeRemoved:
		return "No longer blocked"
	case diff.DiffTypeModified:
		return "Rules changed"
	}
	return ""
}

func (i diffItem) FilterValue() string {
	return i.Title()
}

type DiffModel struct {
	list     list.Model
	diff     *diff.DiffReport
	entries  []diff.DiffEntry
	selected int
	width    int
	height   int
}

func NewDiffModel(diffReport *diff.DiffReport) DiffModel {
	// Combine all entries
	entries := []diff.DiffEntry{}
	entries = append(entries, diffReport.Added...)
	entries = append(entries, diffReport.Removed...)
	entries = append(entries, diffReport.Modified...)

	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = diffItem{entry: e}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = fmt.Sprintf("Differences (%d)", len(entries))
	l.SetShowStatusBar(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("q"),
				key.WithHelp("q", "quit"),
			),
		}
	}

	return DiffModel{
		list:    l,
		diff:    diffReport,
		entries: entries,
	}
}

func (m DiffModel) Init() tea.Cmd {
	return nil
}

func (m DiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width/3-2, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.selected = m.list.Index()
	return m, cmd
}

func (m DiffModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	panelWidth := m.width/3 - 2

	leftStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	centerStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	rightStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

	leftPanel := leftStyle.Render(m.list.View())
	centerPanel := centerStyle.Render(m.renderAnalysis1())
	rightPanel := rightStyle.Render(m.renderAnalysis2())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel, rightPanel)
}

func (m DiffModel) renderAnalysis1() string {
	if m.selected >= len(m.entries) {
		return "No selection"
	}

	entry := m.entries[m.selected]
	if entry.Result1 == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(not blocked)")
	}

	return renderResult(entry.Result1, "Analysis 1")
}

func (m DiffModel) renderAnalysis2() string {
	if m.selected >= len(m.entries) {
		return "No selection"
	}

	entry := m.entries[m.selected]
	if entry.Result2 == nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(not blocked)")
	}

	return renderResult(entry.Result2, "Analysis 2")
}

func renderResult(result *models.Result, title string) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(title))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("%s %s\n", result.Request.Method, result.Request.URL))
	
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Rules:"))
	b.WriteString("\n")
	for _, rule := range result.RulesTriggered {
		b.WriteString(fmt.Sprintf("  • Rule %s: %s\n", rule.ID, rule.Message))
	}

	return b.String()
}
