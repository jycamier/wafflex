package models

import "time"

type HTTPRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	ClientIP string           `json:"client_ip,omitempty"`
}

type RuleTriggered struct {
	ID       string   `json:"id"`
	Message  string   `json:"msg"`
	Severity string   `json:"severity"`
	Tags     []string `json:"tags"`
}

type Interruption struct {
	Action string `json:"action"`
	Status int    `json:"status"`
}

type Result struct {
	ID              string           `json:"id"`
	Request         HTTPRequest      `json:"request"`
	Blocked         bool             `json:"blocked"`
	RulesTriggered  []RuleTriggered  `json:"rules_triggered"`
	Interruption    *Interruption    `json:"interruption,omitempty"`
	Logs            string           `json:"logs"`
}

type Metadata struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalRequests   int       `json:"total_requests"`
	BlockedRequests int       `json:"blocked_requests"`
	TrafficHash     string    `json:"traffic_hash,omitempty"`
	RulesHash       string    `json:"rules_hash,omitempty"`
}

type AnalysisReport struct {
	Metadata Metadata `json:"metadata"`
	Results  []Result `json:"results"`
}
