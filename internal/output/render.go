package output

import (
	"fmt"
	"strings"

	"github.com/ebalyakin/pgkernel/internal/checker"
)

type RenderOptions struct {
	NoColor        bool
	SeverityFilter string
	Share          bool
}

func Render(format string, report checker.Report, options RenderOptions) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch normalized {
	case "pretty", "":
		return RenderPretty(report, options), nil
	case "json":
		return RenderJSON(report)
	case "markdown":
		return RenderMarkdown(report, options), nil
	default:
		return "", fmt.Errorf("unknown output format: %s", format)
	}
}

func filterBySeverity(results []checker.CheckResult, severity string) []checker.CheckResult {
	mode := strings.ToLower(strings.TrimSpace(severity))
	if mode == "" || mode == "all" {
		return results
	}

	filtered := make([]checker.CheckResult, 0, len(results))
	for _, result := range results {
		switch mode {
		case "warn":
			if result.Status == checker.StatusWarn || result.Status == checker.StatusCrit {
				filtered = append(filtered, result)
			}
		case "crit":
			if result.Status == checker.StatusCrit {
				filtered = append(filtered, result)
			}
		default:
			filtered = append(filtered, result)
		}
	}
	return filtered
}
