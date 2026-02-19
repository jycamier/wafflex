package parser

import (
	"bytes"
	"testing"
)

func TestParseHTTPLines(t *testing.T) {
	lines := []string{
		"GET /test HTTP/1.1",
		"Host: example.com",
	}

	req := parseHTTPLines(lines)
	if req == nil {
		t.Fatal("expected non-nil request")
	}

	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("Expected path /test, got %s", req.URL.Path)
	}

	if req.Host != "example.com" {
		t.Errorf("Expected host example.com, got %s", req.Host)
	}
}

func TestParseHTTPLinesWithBody(t *testing.T) {
	body := "param=value"
	lines := []string{
		"POST /api HTTP/1.1",
		"Host: example.com",
		"Content-Length: 11",
		"",
		body,
	}

	req := parseHTTPLines(lines)
	if req == nil {
		t.Fatal("expected non-nil request")
	}

	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(req.Body)
	if buf.String() != body {
		t.Errorf("Expected body %s, got %s", body, buf.String())
	}
}
