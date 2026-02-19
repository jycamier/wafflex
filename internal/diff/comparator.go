package diff

import (
	"crypto/sha256"
	"fmt"

	"github.com/jycamier/wafflex/internal/models"
)

type DiffType string

const (
	DiffTypeAdded    DiffType = "added"    // New blocked request
	DiffTypeRemoved  DiffType = "removed"  // No longer blocked
	DiffTypeModified DiffType = "modified" // Different rules triggered
)

type DiffEntry struct {
	Type     DiffType
	Result1  *models.Result // nil for added
	Result2  *models.Result // nil for removed
	Hash     string
}

type DiffReport struct {
	Added    []DiffEntry
	Removed  []DiffEntry
	Modified []DiffEntry
}

func Compare(report1, report2 *models.AnalysisReport) *DiffReport {
	// Index results by hash
	index1 := make(map[string]*models.Result)
	index2 := make(map[string]*models.Result)

	for i := range report1.Results {
		hash := hashRequest(&report1.Results[i].Request)
		index1[hash] = &report1.Results[i]
	}

	for i := range report2.Results {
		hash := hashRequest(&report2.Results[i].Request)
		index2[hash] = &report2.Results[i]
	}

	diff := &DiffReport{
		Added:    []DiffEntry{},
		Removed:  []DiffEntry{},
		Modified: []DiffEntry{},
	}

	// Find added and modified
	for hash, result2 := range index2 {
		if result1, exists := index1[hash]; exists {
			// Check if rules changed
			if !rulesEqual(result1, result2) {
				diff.Modified = append(diff.Modified, DiffEntry{
					Type:    DiffTypeModified,
					Result1: result1,
					Result2: result2,
					Hash:    hash,
				})
			}
		} else {
			// New blocked request
			diff.Added = append(diff.Added, DiffEntry{
				Type:    DiffTypeAdded,
				Result1: nil,
				Result2: result2,
				Hash:    hash,
			})
		}
	}

	// Find removed
	for hash, result1 := range index1 {
		if _, exists := index2[hash]; !exists {
			diff.Removed = append(diff.Removed, DiffEntry{
				Type:    DiffTypeRemoved,
				Result1: result1,
				Result2: nil,
				Hash:    hash,
			})
		}
	}

	return diff
}

func hashRequest(req *models.HTTPRequest) string {
	data := fmt.Sprintf("%s|%s|%s", req.Method, req.URL, req.Body)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func rulesEqual(r1, r2 *models.Result) bool {
	if len(r1.RulesTriggered) != len(r2.RulesTriggered) {
		return false
	}

	rules1 := make(map[string]bool)
	for _, rule := range r1.RulesTriggered {
		rules1[rule.ID] = true
	}

	for _, rule := range r2.RulesTriggered {
		if !rules1[rule.ID] {
			return false
		}
	}

	return true
}

func (d *DiffReport) Total() int {
	return len(d.Added) + len(d.Removed) + len(d.Modified)
}
