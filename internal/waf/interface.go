package waf

import (
	"net/http"

	"github.com/jycamier/wafflex/internal/models"
)

// WAFEngine is an interface for different WAF implementations
type WAFEngine interface {
	// ProcessRequest processes an HTTP request and returns the analysis result
	ProcessRequest(req *http.Request) (*models.Result, error)
	
	// Name returns the name of the WAF engine
	Name() string
	
	// Version returns the version of the WAF engine
	Version() string
}
