package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ebalyakin/pgkernel/internal/checker"
)

// FILE:internal/output/share.go
// VERSION:1.0.1
// START_MODULE_CONTRACT:
// PURPOSE:Generate compact social-friendly markdown snippets for top risks and fixes.
// SCOPE:--share output used in GitHub issues, Slack, and posts.
// INPUT:Full checker report.
// OUTPUT:Markdown snippet.
// KEYWORDS:[DOMAIN(Growth): virality loop; CONCEPT(Actionability): copy-paste fixes]
// LINKS:[READS_DATA_FROM(checks): critical and warning rows]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.1 - Optimized share block to compact top-3 risk summary for Slack/GitHub snippets.
// PREV_CHANGE_SUMMARY:1.0.0 - Added share block renderer for top-risk communication.
// END_CHANGE_SUMMARY

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
