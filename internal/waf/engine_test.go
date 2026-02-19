package waf

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/corazawaf/coraza/v3"
)

func TestProcessRequestXSS(t *testing.T) {
	// Create engine with basic XSS rule
	cfg := `
SecRuleEngine On
SecRule ARGS "@contains <script>" "id:1,phase:2,deny,status:403,msg:'XSS Attack'"
`
	engine, err := createEngineWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create malicious request
	req, _ := http.NewRequest("GET", "http://example.com/test?param=<script>alert(1)</script>", nil)
	req.Header.Set("Host", "example.com")

	result, err := engine.ProcessRequest(req)
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	if !result.Blocked {
		t.Error("Expected request to be blocked")
	}

	if len(result.RulesTriggered) == 0 {
		t.Error("Expected at least one rule to be triggered")
	}

	// Check if XSS rule was triggered
	found := false
	for _, rule := range result.RulesTriggered {
		if strings.Contains(rule.Message, "XSS") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected XSS rule to be triggered")
	}
}

func TestProcessRequestClean(t *testing.T) {
	cfg := `
SecRuleEngine On
SecRule ARGS "@contains <script>" "id:1,phase:2,deny,status:403,msg:'XSS Attack'"
`
	engine, err := createEngineWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create clean request
	req, _ := http.NewRequest("GET", "http://example.com/test?param=hello", nil)
	req.Header.Set("Host", "example.com")

	result, err := engine.ProcessRequest(req)
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	if result.Blocked {
		t.Error("Expected request to pass")
	}

	if len(result.RulesTriggered) > 0 {
		t.Error("Expected no rules to be triggered")
	}
}

func TestProcessRequestPOSTBody(t *testing.T) {
	cfg := `
SecRuleEngine On
SecRule ARGS "@contains malicious" "id:2,phase:2,deny,status:403,msg:'Malicious content'"
`
	engine, err := createEngineWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	body := bytes.NewBufferString("data=malicious")
	req, _ := http.NewRequest("POST", "http://example.com/api?data=malicious", body)
	req.Header.Set("Host", "example.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := engine.ProcessRequest(req)
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	if !result.Blocked {
		t.Error("Expected request to be blocked")
	}
}

func createEngineWithConfig(config string) (*Engine, error) {
	// Create a temporary config for testing
	return &Engine{
		waf: mustCreateWAF(config),
	}, nil
}

func mustCreateWAF(directives string) coraza.WAF {
	cfg := coraza.NewWAFConfig().WithDirectives(directives)
	waf, err := coraza.NewWAF(cfg)
	if err != nil {
		panic(err)
	}
	return waf
}
