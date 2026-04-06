package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

func RenderShareSnippet(report checker.Report) string {
	b := &strings.Builder{}
	b.WriteString("## Share: Top Risks + Fixes\n\n")

	risky := make([]checker.CheckResult, 0)
	for _, item := range report.Checks {
		if item.Status == checker.StatusWarn || item.Status == checker.StatusCrit {
			risky = append(risky, item)
		}
	}

	if len(risky) == 0 {
		b.WriteString("All checks are green. No high-priority remediation required.\n")
		return b.String()
	}

	sort.Slice(risky, func(i, j int) bool {
		if risky[i].ImpactScore == risky[j].ImpactScore {
			return risky[i].ID < risky[j].ID
		}
		return risky[i].ImpactScore > risky[j].ImpactScore
	})

	if len(risky) > 3 {
		risky = risky[:3]
	}

	b.WriteString(fmt.Sprintf("Detected %d actionable risk(s). Showing top %d by impact.\n\n", report.Summary.Warnings+report.Summary.Criticals, len(risky)))
	b.WriteString("### Top risks\n")
	for _, item := range risky {
		b.WriteString(fmt.Sprintf("- **%s (%s)** `%s` — impact %d/100, confidence `%s`\n", item.ID, item.Status, item.Name, item.ImpactScore, item.Confidence))
	}

	b.WriteString("\n### Copy-paste fixes\n")
	for _, item := range risky {
		if strings.TrimSpace(item.Fix) == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- `%s`: `%s`\n", item.ID, item.Fix))
	}

	return b.String()
}
