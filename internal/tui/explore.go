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
	list     list.Model
	viewport viewport.Model
	results  []models.Result
	selected int
	width    int
	height   int
	focused  bool
	ready    bool
	groupBy  string // "", "rule", "ip", "user-agent"
	grouped  map[string][]models.Result
}

func NewExploreModel(results []models.Result) ExploreModel {
	items := make([]list.Item, len(results))
	for i, r := range results {
		items[i] = item{result: r}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Blocked Requests"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "filter by rule/tag/method"),
			),
			key.NewBinding(
				key.WithKeys("↑/↓"),
				key.WithHelp("↑/↓", "navigate"),
			),
			key.NewBinding(
				key.WithKeys("g"),
				key.WithHelp("g", "group by (rule/ip/header)"),
			),
			key.NewBinding(
				key.WithKeys("q"),
				key.WithHelp("q", "quit"),
			),
		}
	}

	return ExploreModel{
		list:    l,
		results: results,
		focused: true,
		groupBy: "",
		grouped: make(map[string][]models.Result),
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
		case "g":
			// Cycle through group by modes
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
			return m, nil
		}
	}

	var cmd tea.Cmd
	
	// Update list
	oldSelected := m.list.Index()
	m.list, cmd = m.list.Update(msg)
	m.selected = m.list.Index()
	
	// Update viewport content if selection changed
	if oldSelected != m.selected {
		m.viewport.SetContent(m.renderDetails())
		m.viewport.GotoTop()
	}
	
	// Update viewport for scrolling
	m.viewport, _ = m.viewport.Update(msg)
	
	return m, cmd
}

func (m ExploreModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	leftWidth := m.width/2 - 2
	rightWidth := m.width/2 - 2

	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.height - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1)

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
		return m.renderResultDetails(item.result)
	}
	
	return "No selection"
}

func (m ExploreModel) renderGroupDetails(group groupItem) string {
	var sections []string
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		MarginBottom(1)
	
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33"))
	
	// Group summary
	var summary strings.Builder
	summary.WriteString(labelStyle.Render("Group:") + " " + group.key + "\n")
	summary.WriteString(labelStyle.Render("Count:") + fmt.Sprintf(" %d requests\n\n", group.count))
	
	// Show first 5 examples
	summary.WriteString(labelStyle.Render("Examples:") + "\n")
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
	
	sections = append(sections, titleStyle.Render("GROUP SUMMARY")+"\n"+summary.String())
	
	return strings.Join(sections, "\n\n")
}

func (m ExploreModel) renderResultDetails(result models.Result) string {
	
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		MarginBottom(1)
	
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33"))
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)
	
	ruleStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginBottom(1)
	
	var sections []string
	
	// HTTP Request section
	var reqContent strings.Builder
	reqContent.WriteString(labelStyle.Render("Method & URL") + "\n")
	reqContent.WriteString(valueStyle.Render(fmt.Sprintf("%s %s", result.Request.Method, result.Request.URL)) + "\n\n")
	
	if result.Request.ClientIP != "" {
		reqContent.WriteString(labelStyle.Render("Client IP") + "\n")
		reqContent.WriteString(valueStyle.Render(result.Request.ClientIP) + "\n\n")
	}
	
	reqContent.WriteString(labelStyle.Render("Headers") + "\n")
	if len(result.Request.Headers) > 0 {
		for k, v := range result.Request.Headers {
			reqContent.WriteString(fmt.Sprintf("  %s: %s\n", k, valueStyle.Render(v)))
		}
	} else {
		reqContent.WriteString(dimStyle.Render("  (no headers captured)") + "\n")
	}
	reqContent.WriteString("\n")
	
	reqContent.WriteString(labelStyle.Render("Body") + "\n")
	if result.Request.Body != "" {
		body := result.Request.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		reqContent.WriteString(valueStyle.Render(body) + "\n")
	} else {
		reqContent.WriteString(dimStyle.Render("  (empty)") + "\n")
	}
	
	sections = append(sections, titleStyle.Render("HTTP REQUEST")+"\n"+reqContent.String())
	
	// Rules section
	var rulesContent strings.Builder
	for i, rule := range result.RulesTriggered {
		var ruleText strings.Builder
		ruleText.WriteString(labelStyle.Render(fmt.Sprintf("Rule #%d: %s", i+1, rule.ID)) + "\n")
		ruleText.WriteString(fmt.Sprintf("Severity: %s\n", rule.Severity))
		ruleText.WriteString(fmt.Sprintf("Message: %s\n", rule.Message))
		if len(rule.Tags) > 0 {
			ruleText.WriteString(fmt.Sprintf("Tags: %s", strings.Join(rule.Tags, ", ")))
		}
		rulesContent.WriteString(ruleStyle.Render(ruleText.String()) + "\n")
	}
	
	sections = append(sections, titleStyle.Render("RULES TRIGGERED")+"\n"+rulesContent.String())
	
	// Interruption section
	if result.Interruption != nil {
		var intContent strings.Builder
		intContent.WriteString(fmt.Sprintf("%s: %s\n", labelStyle.Render("Action"), result.Interruption.Action))
		intContent.WriteString(fmt.Sprintf("%s: %d", labelStyle.Render("Status"), result.Interruption.Status))
		sections = append(sections, titleStyle.Render("INTERRUPTION")+"\n"+intContent.String())
	}
	
	// Logs section
	if result.Logs != "" {
		sections = append(sections, titleStyle.Render("CORAZA LOGS")+"\n"+dimStyle.Render(result.Logs))
	}
	
	return strings.Join(sections, "\n\n")
}

func (m *ExploreModel) rebuildList() {
	var items []list.Item
	m.grouped = make(map[string][]models.Result)
	
	if m.groupBy == "" {
		// No grouping, show all results
		for _, r := range m.results {
			items = append(items, item{result: r})
		}
		m.list.Title = "Blocked Requests"
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
		
		m.list.Title = fmt.Sprintf("Blocked Requests (grouped by %s)", m.groupBy)
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
