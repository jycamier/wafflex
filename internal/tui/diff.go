package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jycamier/wafflex/internal/diff"
	"github.com/jycamier/wafflex/internal/models"
)

var (
	addedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	removedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	modifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

type diffItem struct {
	entry diff.DiffEntry
}

func (i diffItem) Title() string {
	var prefix string
	var req *models.HTTPRequest

	switch i.entry.Type {
	case diff.DiffTypeAdded:
		prefix = addedStyle.Render("[+]")
		req = &i.entry.Result2.Request
	case diff.DiffTypeRemoved:
		prefix = removedStyle.Render("[-]")
		req = &i.entry.Result1.Request
	case diff.DiffTypeModified:
		prefix = modifiedStyle.Render("[~]")
		req = &i.entry.Result1.Request
	}

	return fmt.Sprintf("%s %s %s", prefix, req.Method, req.URL)
}

func (i diffItem) Description() string {
	switch i.entry.Type {
	case diff.DiffTypeAdded:
		return addedStyle.Render("New blocked request")
	case diff.DiffTypeRemoved:
		return removedStyle.Render("No longer blocked")
	case diff.DiffTypeModified:
		return modifiedStyle.Render("Rules changed")
	}
	return ""
}

func (i diffItem) FilterValue() string {
	return i.Title()
}

type diffGroupItem struct {
	key     string
	count   int
	entries []diff.DiffEntry
}

func (i diffGroupItem) Title() string {
	return fmt.Sprintf("[%d] %s", i.count, i.key)
}

func (i diffGroupItem) Description() string {
	return fmt.Sprintf("%d differences", i.count)
}

func (i diffGroupItem) FilterValue() string {
	return i.key
}

// focus: 0=list, 1=panel1, 2=panel2
type DiffModel struct {
	list      list.Model
	vp1       viewport.Model
	vp2       viewport.Model
	diff      *diff.DiffReport
	entries   []diff.DiffEntry
	selected  int
	width     int
	height    int
	ready     bool
	groupBy   string
	focus     int
	title1    string
	title2    string
}

func NewDiffModel(diffReport *diff.DiffReport) DiffModel {
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
	l.SetFilteringEnabled(true)
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel focus")),
			key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "group by (type/rule)")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
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
		panelWidth := msg.Width/3 - 2
		m.list.SetSize(panelWidth, msg.Height-2)

		vpWidth := panelWidth - 4
		vpHeight := msg.Height - 8 // leave room for fixed title
		if !m.ready {
			m.vp1 = viewport.New(vpWidth, vpHeight)
			m.vp2 = viewport.New(vpWidth, vpHeight)
			m.ready = true
		} else {
			m.vp1.Width = vpWidth
			m.vp1.Height = vpHeight
			m.vp2.Width = vpWidth
			m.vp2.Height = vpHeight
		}

		m.updateViewports()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % 3
			return m, nil
		case "g":
			if m.focus == 0 {
				switch m.groupBy {
				case "":
					m.groupBy = "type"
				case "type":
					m.groupBy = "rule"
				case "rule":
					m.groupBy = ""
				}
				m.rebuildList()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd

	switch m.focus {
	case 0:
		oldSelected := m.list.Index()
		m.list, cmd = m.list.Update(msg)
		m.selected = m.list.Index()
		if oldSelected != m.selected {
			m.updateViewports()
		}
	case 1:
		m.vp1, cmd = m.vp1.Update(msg)
	case 2:
		m.vp2, cmd = m.vp2.Update(msg)
	}

	return m, cmd
}

func (m *DiffModel) updateViewports() {
	if !m.ready {
		return
	}
	m.title1, _ = m.panelContent(1)
	_, content1 := m.panelContent(1)
	m.title2, _ = m.panelContent(2)
	_, content2 := m.panelContent(2)

	panelWidth := m.width/3 - 2
	isGroup := m.title2 == ""
	if isGroup {
		m.vp1.Width = m.width - panelWidth - 8
	} else {
		m.vp1.Width = panelWidth - 4
	}

	m.vp1.SetContent(content1)
	m.vp1.GotoTop()
	m.vp2.SetContent(content2)
	m.vp2.GotoTop()
}

func (m DiffModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	panelWidth := m.width/3 - 2
	focusColor := lipgloss.Color("205")
	normalColor := lipgloss.Color("62")

	borderColor := func(panel int) lipgloss.Color {
		if m.focus == panel {
			return focusColor
		}
		return normalColor
	}

	leftStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor(0))

	detailStyle := func(panel int) lipgloss.Style {
		return lipgloss.NewStyle().
			Width(panelWidth).
			Height(m.height - 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor(panel)).
			Padding(0, 1)
	}

	leftPanel := leftStyle.Render(m.list.View())

	if !m.ready {
		return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, detailStyle(1).Render("Loading..."))
	}

	isGroup := m.title2 == ""
	if isGroup {
		// Group summary: single wide detail panel
		wideStyle := lipgloss.NewStyle().
			Width(m.width - panelWidth - 4).
			Height(m.height - 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor(1)).
			Padding(0, 1)
		header := detailTitleStyle.Render(m.title1)
		return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, wideStyle.Render(header+"\n\n"+m.vp1.View()))
	}

	header1 := detailTitleStyle.Render(m.title1)
	header2 := detailTitleStyle.Render(m.title2)
	centerPanel := detailStyle(1).Render(header1 + "\n\n" + m.vp1.View())
	rightPanel := detailStyle(2).Render(header2 + "\n\n" + m.vp2.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, centerPanel, rightPanel)
}

// panelContent returns (fixedTitle, scrollableContent) for panel 1 or 2.
func (m DiffModel) panelContent(panel int) (string, string) {
	items := m.list.Items()
	if m.selected >= len(items) {
		return "", "No selection"
	}

	sel := items[m.selected]

	if g, ok := sel.(diffGroupItem); ok {
		if panel == 1 {
			return "GROUP SUMMARY", m.renderGroupSummary(g)
		}
		return "", ""
	}

	if d, ok := sel.(diffItem); ok {
		title := fmt.Sprintf("Analysis %d", panel)
		var result *models.Result
		if panel == 1 {
			result = d.entry.Result1
		} else {
			result = d.entry.Result2
		}
		if result == nil {
			return title, detailDimStyle.Render("Request was not blocked in this analysis.\n\nNo rules were triggered and the request passed through the WAF.")
		}
		return title, RenderResultDetail(result, "")
	}

	return "", "No selection"
}

func (m DiffModel) renderGroupSummary(g diffGroupItem) string {
	var b strings.Builder
	b.WriteString(detailLabelStyle.Render("Group:") + " " + g.key + "\n")
	b.WriteString(detailLabelStyle.Render("Count:") + fmt.Sprintf(" %d differences\n\n", g.count))

	b.WriteString(detailLabelStyle.Render("Entries:") + "\n")
	for i, entry := range g.entries {
		if i >= 5 {
			b.WriteString(fmt.Sprintf("\n... and %d more", len(g.entries)-5))
			break
		}
		var prefix, url string
		switch entry.Type {
		case diff.DiffTypeAdded:
			prefix = addedStyle.Render("[+]")
			url = fmt.Sprintf("%s %s", entry.Result2.Request.Method, entry.Result2.Request.URL)
		case diff.DiffTypeRemoved:
			prefix = removedStyle.Render("[-]")
			url = fmt.Sprintf("%s %s", entry.Result1.Request.Method, entry.Result1.Request.URL)
		case diff.DiffTypeModified:
			prefix = modifiedStyle.Render("[~]")
			url = fmt.Sprintf("%s %s", entry.Result1.Request.Method, entry.Result1.Request.URL)
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", prefix, url))
	}

	return b.String()
}

func (m *DiffModel) rebuildList() {
	var items []list.Item

	if m.groupBy == "" {
		for _, e := range m.entries {
			items = append(items, diffItem{entry: e})
		}
		m.list.Title = fmt.Sprintf("Differences (%d)", len(m.entries))
	} else {
		groups := make(map[string][]diff.DiffEntry)
		for _, e := range m.entries {
			var key string
			switch m.groupBy {
			case "type":
				switch e.Type {
				case diff.DiffTypeAdded:
					key = "Added"
				case diff.DiffTypeRemoved:
					key = "Removed"
				case diff.DiffTypeModified:
					key = "Modified"
				}
			case "rule":
				r := e.Result1
				if r == nil {
					r = e.Result2
				}
				if r != nil && len(r.RulesTriggered) > 0 {
					key = fmt.Sprintf("Rule %s: %s", r.RulesTriggered[0].ID, r.RulesTriggered[0].Message)
				} else {
					key = "No rules"
				}
			}
			groups[key] = append(groups[key], e)
		}

		for key, entries := range groups {
			items = append(items, diffGroupItem{key: key, count: len(entries), entries: entries})
		}
		m.list.Title = fmt.Sprintf("Differences (grouped by %s)", m.groupBy)
	}

	m.list.SetItems(items)
	m.selected = 0
	m.updateViewports()
}
