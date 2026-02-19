package waf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jycamier/wafflex/internal/models"
)

// CustomEngine is a simple rule-based WAF engine using JSON configuration
type CustomEngine struct {
	rules []CustomRule
}

// Ensure CustomEngine implements WAFEngine
var _ WAFEngine = (*CustomEngine)(nil)

// CustomRule represents a simple detection rule
type CustomRule struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Pattern  string   `json:"pattern"`
	Location string   `json:"location"` // "url", "body", "headers", "all"
	Action   string   `json:"action"`   // "block", "log"
	Severity string   `json:"severity"`
	Tags     []string `json:"tags"`
	compiled *regexp.Regexp
}

// CustomConfig represents the JSON configuration
type CustomConfig struct {
	Engine string       `json:"engine"`
	Rules  []CustomRule `json:"rules"`
}

func NewCustomEngine(configPath string) (*CustomEngine, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	var config CustomConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Compile regex patterns
	for i := range config.Rules {
		compiled, err := regexp.Compile(config.Rules[i].Pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile pattern for rule %s: %w", config.Rules[i].ID, err)
		}
		config.Rules[i].compiled = compiled
	}

	return &CustomEngine{
		rules: config.Rules,
	}, nil
}

func (c *CustomEngine) ProcessRequest(req *http.Request) (*models.Result, error) {
	result := &models.Result{
		ID: uuid.New().String(),
		Request: models.HTTPRequest{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: make(map[string]string),
			Body:    "",
		},
		Blocked:        false,
		RulesTriggered: []models.RuleTriggered{},
	}

	// Copy headers
	for k, v := range req.Header {
		result.Request.Headers[k] = strings.Join(v, ", ")
	}

	// Extract client IP
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		result.Request.ClientIP = strings.Split(xff, ",")[0]
	} else if req.RemoteAddr != "" {
		result.Request.ClientIP = req.RemoteAddr
	}

	// Read body
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		result.Request.Body = string(bodyBytes)
	}

	// Check each rule
	var logs strings.Builder
	for _, rule := range c.rules {
		matched := false

		switch rule.Location {
		case "url":
			matched = rule.compiled.MatchString(req.URL.String())
		case "body":
			matched = rule.compiled.MatchString(result.Request.Body)
		case "headers":
			for _, v := range result.Request.Headers {
				if rule.compiled.MatchString(v) {
					matched = true
					break
				}
			}
		case "all":
			// Check URL, body, and headers
			matched = rule.compiled.MatchString(req.URL.String()) ||
				rule.compiled.MatchString(result.Request.Body)
			if !matched {
				for _, v := range result.Request.Headers {
					if rule.compiled.MatchString(v) {
						matched = true
						break
					}
				}
			}
		}

		if matched {
			result.RulesTriggered = append(result.RulesTriggered, models.RuleTriggered{
				ID:       rule.ID,
				Message:  rule.Name,
				Severity: rule.Severity,
				Tags:     rule.Tags,
			})

			logs.WriteString(fmt.Sprintf("[%s] Rule %s: %s\n", rule.Severity, rule.ID, rule.Name))

			if rule.Action == "block" {
				result.Blocked = true
				result.Interruption = &models.Interruption{
					Action: "deny",
					Status: 403,
				}
			}
		}
	}

	result.Logs = logs.String()

	return result, nil
}

func (c *CustomEngine) Name() string {
	return "Custom"
}

func (c *CustomEngine) Version() string {
	return "1.0"
}
