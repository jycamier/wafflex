package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jycamier/wafflex/internal/models"
)

var (
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("235")).
				Padding(0, 1).
				MarginBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("33"))

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	detailDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	detailRuleStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginBottom(1)
)

// RenderResultDetail renders the detail panel for a single analysis result.
func RenderResultDetail(result *models.Result, title string) string {
	var sections []string

	// HTTP Request section
	var req strings.Builder
	req.WriteString(detailLabelStyle.Render("Method & URL") + "\n")
	req.WriteString(detailValueStyle.Render(fmt.Sprintf("%s %s", result.Request.Method, result.Request.URL)) + "\n\n")

	if result.Request.ClientIP != "" {
		req.WriteString(detailLabelStyle.Render("Client IP") + "\n")
		req.WriteString(detailValueStyle.Render(result.Request.ClientIP) + "\n\n")
	}

	req.WriteString(detailLabelStyle.Render("Headers") + "\n")
	if len(result.Request.Headers) > 0 {
		for k, v := range result.Request.Headers {
			req.WriteString(fmt.Sprintf("  %s: %s\n", k, detailValueStyle.Render(v)))
		}
	} else {
		req.WriteString(detailDimStyle.Render("  (no headers captured)") + "\n")
	}
	req.WriteString("\n")

	req.WriteString(detailLabelStyle.Render("Body") + "\n")
	if result.Request.Body != "" {
		body := result.Request.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		req.WriteString(detailValueStyle.Render(body) + "\n")
	} else {
		req.WriteString(detailDimStyle.Render("  (empty)") + "\n")
	}

	reqTitle := "HTTP REQUEST"
	if title != "" {
		reqTitle = title
	}
	sections = append(sections, detailTitleStyle.Render(reqTitle)+"\n"+req.String())

	// Rules section
	var rules strings.Builder
	for i, rule := range result.RulesTriggered {
		var rt strings.Builder
		rt.WriteString(detailLabelStyle.Render(fmt.Sprintf("Rule #%d: %s", i+1, rule.ID)) + "\n")
		rt.WriteString(fmt.Sprintf("Severity: %s\n", rule.Severity))
		rt.WriteString(fmt.Sprintf("Message: %s\n", rule.Message))
		if len(rule.Tags) > 0 {
			rt.WriteString(fmt.Sprintf("Tags: %s", strings.Join(rule.Tags, ", ")))
		}
		rules.WriteString(detailRuleStyle.Render(rt.String()) + "\n")
	}
	sections = append(sections, detailTitleStyle.Render("RULES TRIGGERED")+"\n"+rules.String())

	// Interruption section
	if result.Interruption != nil {
		var intContent strings.Builder
		intContent.WriteString(fmt.Sprintf("%s: %s\n", detailLabelStyle.Render("Action"), result.Interruption.Action))
		intContent.WriteString(fmt.Sprintf("%s: %d", detailLabelStyle.Render("Status"), result.Interruption.Status))
		sections = append(sections, detailTitleStyle.Render("INTERRUPTION")+"\n"+intContent.String())
	}

	// Logs section
	if result.Logs != "" {
		sections = append(sections, detailTitleStyle.Render("CORAZA LOGS")+"\n"+detailDimStyle.Render(result.Logs))
	}

	return strings.Join(sections, "\n\n")
}

