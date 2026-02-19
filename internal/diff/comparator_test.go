package diff

import (
	"testing"

	"github.com/jycamier/wafflex/internal/models"
)

func TestCompareAdded(t *testing.T) {
	report1 := &models.AnalysisReport{
		Results: []models.Result{
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test1", Body: ""},
				Blocked: true,
			},
		},
	}

	report2 := &models.AnalysisReport{
		Results: []models.Result{
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test1", Body: ""},
				Blocked: true,
			},
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test2", Body: ""},
				Blocked: true,
			},
		},
	}

	diff := Compare(report1, report2)

	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added, got %d", len(diff.Added))
	}

	if len(diff.Removed) != 0 {
		t.Errorf("Expected 0 removed, got %d", len(diff.Removed))
	}
}

func TestCompareRemoved(t *testing.T) {
	report1 := &models.AnalysisReport{
		Results: []models.Result{
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test1", Body: ""},
				Blocked: true,
			},
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test2", Body: ""},
				Blocked: true,
			},
		},
	}

	report2 := &models.AnalysisReport{
		Results: []models.Result{
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test1", Body: ""},
				Blocked: true,
			},
		},
	}

	diff := Compare(report1, report2)

	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed, got %d", len(diff.Removed))
	}

	if len(diff.Added) != 0 {
		t.Errorf("Expected 0 added, got %d", len(diff.Added))
	}
}

func TestCompareModified(t *testing.T) {
	report1 := &models.AnalysisReport{
		Results: []models.Result{
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test1", Body: ""},
				Blocked: true,
				RulesTriggered: []models.RuleTriggered{
					{ID: "1", Message: "Rule 1"},
				},
			},
		},
	}

	report2 := &models.AnalysisReport{
		Results: []models.Result{
			{
				Request: models.HTTPRequest{Method: "GET", URL: "/test1", Body: ""},
				Blocked: true,
				RulesTriggered: []models.RuleTriggered{
					{ID: "1", Message: "Rule 1"},
					{ID: "2", Message: "Rule 2"},
				},
			},
		},
	}

	diff := Compare(report1, report2)

	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified, got %d", len(diff.Modified))
	}
}
