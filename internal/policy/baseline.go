package policy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/balyakin/pgkernel/internal/checker"
)

func LoadReport(path string) (checker.Report, error) {
	var report checker.Report
	raw, err := os.ReadFile(path)
	if err != nil {
		return report, fmt.Errorf("read baseline report: %w", err)
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return report, fmt.Errorf("parse baseline report: %w", err)
	}
	return report, nil
}

func DetectRegressions(current []checker.CheckResult, previous checker.Report) []checker.Regression {
	previousByID := make(map[string]checker.Status, len(previous.Checks))
	for _, item := range previous.Checks {
		previousByID[item.ID] = item.Status
	}

	regressions := make([]checker.Regression, 0)
	for _, item := range current {
		prev, ok := previousByID[item.ID]
		if !ok {
			continue
		}

		if statusWeight(item.Status) > statusWeight(prev) {
			regressions = append(regressions, checker.Regression{
				ID:           item.ID,
				Current:      item.Status,
				Previous:     prev,
				Description:  fmt.Sprintf("Status worsened from %s to %s", prev, item.Status),
				SeverityBump: isSeverityBump(prev, item.Status),
			})
		}
	}

	return regressions
}

func statusWeight(status checker.Status) int {
	switch status {
	case checker.StatusCrit:
		return 3
	case checker.StatusWarn:
		return 2
	case checker.StatusInfo:
		return 1
	case checker.StatusSkip:
		return 0
	case checker.StatusPass:
		return 0
	default:
		return 0
	}
}

func isSeverityBump(previous checker.Status, current checker.Status) bool {
	if current != checker.StatusCrit {
		return false
	}
	return previous != checker.StatusCrit
}
