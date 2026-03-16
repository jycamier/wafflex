package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jycamier/wafflex/internal/models"
)

type item struct {
	result models.Result
}

func (i item) Title() string {
	return fmt.Sprintf("%s %s", i.result.Request.Method, i.result.Request.URL)
}

func (i item) Description() string {
	if len(i.result.RulesTriggered) > 0 {
		return fmt.Sprintf("Rules: %s", i.result.RulesTriggered[0].Message)
	}
	return "No rules triggered"
}

func (i item) FilterValue() string {
	tags := []string{}
	for _, rule := range i.result.RulesTriggered {
		tags = append(tags, rule.Tags...)
	}
	return fmt.Sprintf("%s %s %s", i.result.Request.Method, i.result.Request.URL, strings.Join(tags, " "))
}

type ExploreModel struct {
	list          list.Model
	viewport      viewport.Model
	results       []models.Result
	totalRequests int
	selected      int
	width         int
	height        int
	focused       bool
	ready         bool
	groupBy       string // "", "rule", "ip", "user-agent"
	grouped       map[string][]models.Result
}

func NewExploreModel(results []models.Result, totalRequests int) ExploreModel {
	items := make([]list.Item, len(results))
	for i, r := range results {
		items[i] = item{result: r}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	pct := float64(0)
	if totalRequests > 0 {
		pct = float64(len(results)) / float64(totalRequests) * 100
	}
	l.Title = fmt.Sprintf("Blocked Requests — %.1f%% blocked (%d/%d)", pct, len(results), totalRequests)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel focus")),
			key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
			key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "group by")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	}

	return ExploreModel{
		list:          l,
		results:       results,
		totalRequests: totalRequests,
		focused:       false,
		groupBy:       "",
		grouped:       make(map[string][]models.Result),
	}
}

func (m ExploreModel) Init() tea.Cmd {
	return nil
}

func (m ExploreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width/2-2, msg.Height-2)
		
		if !m.ready {
			m.viewport = viewport.New(msg.Width/2-4, msg.Height-6)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width/2 - 4
			m.viewport.Height = msg.Height - 6
		}
		
		m.viewport.SetContent(m.renderDetails())
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focused = !m.focused
			return m, nil
		case "g":
			if !m.focused {
				switch m.groupBy {
				case "":
					m.groupBy = "rule"
				case "rule":
					m.groupBy = "ip"
				case "ip":
					m.groupBy = "user-agent"
				case "user-agent":
					m.groupBy = ""
				}
				m.rebuildList()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd

	if m.focused {
		m.viewport, cmd = m.viewport.Update(msg)
	} else {
		oldSelected := m.list.Index()
		m.list, cmd = m.list.Update(msg)
		m.selected = m.list.Index()

		if oldSelected != m.selected {
			m.viewport.SetContent(m.renderDetails())
			m.viewport.GotoTop()
		}
	}

	return m, cmd
}

func (m ExploreModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	leftWidth := m.width/2 - 2
	rightWidth := m.width/2 - 2
	focusColor := lipgloss.Color("205")
	normalColor := lipgloss.Color("62")

	leftBorder := normalColor
	rightBorder := normalColor
	if m.focused {
		rightBorder = focusColor
	} else {
		leftBorder = focusColor
	}

	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorder)

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorder).
		Padding(0, 1)

	leftPanel := leftStyle.Render(m.list.View())

	var rightPanel string
	if m.ready {
		rightPanel = rightStyle.Render(m.viewport.View())
	} else {
		rightPanel = rightStyle.Render("Loading...")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m ExploreModel) renderDetails() string {
	items := m.list.Items()
	if m.selected >= len(items) {
		return "No selection"
	}
	
	selectedItem := items[m.selected]
	
	// Check if it's a group item
	if groupItem, ok := selectedItem.(groupItem); ok {
		return m.renderGroupDetails(groupItem)
	}
	
	// Regular item
	if item, ok := selectedItem.(item); ok {
		return RenderResultDetail(&item.result, "HTTP REQUEST")
	}
	
	return "No selection"
}

func (m ExploreModel) renderGroupDetails(group groupItem) string {
	var sections []string

	// Group summary
	var summary strings.Builder
	summary.WriteString(detailLabelStyle.Render("Group:") + " " + group.key + "\n")
	summary.WriteString(detailLabelStyle.Render("Count:") + fmt.Sprintf(" %d requests\n\n", group.count))
	
	// Show first 5 examples
	summary.WriteString(detailLabelStyle.Render("Examples:") + "\n")
	for i, result := range group.results {
		if i >= 5 {
			summary.WriteString(fmt.Sprintf("\n... and %d more", len(group.results)-5))
			break
		}
		summary.WriteString(fmt.Sprintf("  • %s %s", result.Request.Method, result.Request.URL))
		if result.Request.ClientIP != "" {
			summary.WriteString(fmt.Sprintf(" [%s]", result.Request.ClientIP))
		}
		summary.WriteString("\n")
	}
	
	sections = append(sections, detailTitleStyle.Render("GROUP SUMMARY")+"\n"+summary.String())
	
	return strings.Join(sections, "\n\n")
}

func (m ExploreModel) statsTitle(suffix string) string {
	pct := float64(0)
	if m.totalRequests > 0 {
		pct = float64(len(m.results)) / float64(m.totalRequests) * 100
	}
	title := fmt.Sprintf("Blocked Requests — %.1f%% blocked (%d/%d)", pct, len(m.results), m.totalRequests)
	if suffix != "" {
		title += " — " + suffix
	}
	return title
}

func (m *ExploreModel) rebuildList() {
	var items []list.Item
	m.grouped = make(map[string][]models.Result)
	
	if m.groupBy == "" {
		// No grouping, show all results
		for _, r := range m.results {
			items = append(items, item{result: r})
		}
		m.list.Title = m.statsTitle("")
	} else {
		// Group results
		for _, r := range m.results {
			var key string
			switch m.groupBy {
			case "rule":
				if len(r.RulesTriggered) > 0 {
					key = fmt.Sprintf("Rule %s: %s", r.RulesTriggered[0].ID, r.RulesTriggered[0].Message)
				} else {
					key = "No rules"
				}
			case "ip":
				if r.Request.ClientIP != "" {
					key = r.Request.ClientIP
				} else {
					key = "Unknown IP"
				}
			case "user-agent":
				if ua := r.Request.Headers["User-Agent"]; ua != "" {
					// Truncate long user agents
					if len(ua) > 50 {
						key = ua[:50] + "..."
					} else {
						key = ua
					}
				} else {
					key = "No User-Agent"
				}
			}
			m.grouped[key] = append(m.grouped[key], r)
		}
		
		// Create items with counts
		for key, results := range m.grouped {
			// Use first result as representative
			items = append(items, groupItem{
				key:     key,
				count:   len(results),
				results: results,
			})
		}
		
		m.list.Title = m.statsTitle(fmt.Sprintf("grouped by %s", m.groupBy))
	}
	
	m.list.SetItems(items)
	m.selected = 0
	if m.ready {
		m.viewport.SetContent(m.renderDetails())
		m.viewport.GotoTop()
	}
}

type groupItem struct {
	key     string
	count   int
	results []models.Result
}

func (i groupItem) Title() string {
	return fmt.Sprintf("[%d] %s", i.count, i.key)
}

func (i groupItem) Description() string {
	return fmt.Sprintf("%d blocked requests", i.count)
}

func (i groupItem) FilterValue() string {
	return i.key
}
