package waf

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/corazawaf/coraza/v3"
	"github.com/google/uuid"
	"github.com/jycamier/wafflex/internal/models"
)

type Engine struct {
	waf coraza.WAF
}

// Ensure Engine implements WAFEngine
var _ WAFEngine = (*Engine)(nil)

func NewEngine(configPath string) (*Engine, error) {
	cfg := coraza.NewWAFConfig()
	
	// Load configuration from file
	if configPath != "" {
		cfg = cfg.WithDirectivesFromFile(configPath)
	}

	waf, err := coraza.NewWAF(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAF: %w", err)
	}

	return &Engine{waf: waf}, nil
}

func (e *Engine) ProcessRequest(req *http.Request) (*models.Result, error) {
	tx := e.waf.NewTransaction()
	defer func() {
		tx.ProcessLogging()
		tx.Close()
	}()

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
	
	// Extract client IP from X-Forwarded-For header
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		result.Request.ClientIP = strings.Split(xff, ",")[0]
	} else if req.RemoteAddr != "" {
		result.Request.ClientIP = req.RemoteAddr
	}

	// Read body
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		result.Request.Body = string(bodyBytes)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Process connection
	tx.ProcessConnection(req.RemoteAddr, 0, "127.0.0.1", 80)

	// Process URI
	tx.ProcessURI(req.URL.String(), req.Method, req.Proto)

	// Add headers
	for k, v := range req.Header {
		for _, val := range v {
			tx.AddRequestHeader(k, val)
		}
	}

	// Process request headers (Phase 1)
	if it := tx.ProcessRequestHeaders(); it != nil {
		result.Blocked = true
		result.Interruption = &models.Interruption{
			Action: it.Action,
			Status: it.Status,
		}
	}

	// Write request body
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		if len(bodyBytes) > 0 {
			_, _, _ = tx.WriteRequestBody(bodyBytes)
		}
	}

	// Process request body (Phase 2)
	if it, err := tx.ProcessRequestBody(); err == nil && it != nil {
		result.Blocked = true
		result.Interruption = &models.Interruption{
			Action: it.Action,
			Status: it.Status,
		}
	}

	// Collect triggered rules
	for _, rule := range tx.MatchedRules() {
		result.RulesTriggered = append(result.RulesTriggered, models.RuleTriggered{
			ID:       fmt.Sprintf("%d", rule.Rule().ID()),
			Message:  rule.Message(),
			Severity: severityToString(int(rule.Rule().Severity())),
			Tags:     rule.Rule().Tags(),
		})
	}

	// Collect logs
	var logBuffer strings.Builder
	for _, rule := range tx.MatchedRules() {
		logBuffer.WriteString(fmt.Sprintf("[%s] Rule %d: %s\n", 
			severityToString(int(rule.Rule().Severity())),
			rule.Rule().ID(),
			rule.Message()))
	}
	result.Logs = logBuffer.String()

	return result, nil
}

func severityToString(sev int) string {
	switch sev {
	case 0:
		return "EMERGENCY"
	case 1:
		return "ALERT"
	case 2:
		return "CRITICAL"
	case 3:
		return "ERROR"
	case 4:
		return "WARNING"
	case 5:
		return "NOTICE"
	case 6:
		return "INFO"
	case 7:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func (e *Engine) Name() string {
	return "Coraza"
}

func (e *Engine) Version() string {
	return "v3"
}
