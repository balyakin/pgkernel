package output

import (
	"fmt"
	"strings"

	"github.com/ebalyakin/pgkernel/internal/checker"
)

// FILE:internal/output/markdown.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Render markdown report suitable for GitHub issues and Slack docs.
// SCOPE:Summary header, checks table, optional share block.
// INPUT:checker.Report and rendering options.
// OUTPUT:Markdown text.
// KEYWORDS:[DOMAIN(Collaboration): shareable reports; CONCEPT(Virality): copy-paste snippets]
// LINKS:[USES_API(strings.Builder): markdown assembly]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added markdown renderer.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

func RenderMarkdown(report checker.Report, options RenderOptions) string {
	checks := filterBySeverity(report.Checks, options.SeverityFilter)
	b := &strings.Builder{}

	b.WriteString(fmt.Sprintf("# pgkernel v%s Report\n\n", report.Version))
	b.WriteString(fmt.Sprintf("- **Profile:** %s\n", report.Profile))
	b.WriteString(fmt.Sprintf("- **System:** %s %s | %s\n", report.System.Kernel, report.System.Arch, report.System.Distro))
	b.WriteString(fmt.Sprintf("- **PostgreSQL:** %s | `%s`\n", report.PostgreSQL.Version, fallbackString(report.PostgreSQL.DataDir, "unknown")))
	b.WriteString(fmt.Sprintf("- **Summary:** total=%d passed=%d warn=%d crit=%d skipped=%d exit=%d\n\n", report.Summary.Total, report.Summary.Passed, report.Summary.Warnings, report.Summary.Criticals, report.Summary.Skipped, report.Summary.ExitCode))

	b.WriteString("| ID | Category | Status | Impact | Confidence | Current | Expected | Fix |\n")
	b.WriteString("|---|---|---|---:|---|---|---|---|\n")
	for _, check := range checks {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s | %s | %s | %s |\n",
			check.ID,
			check.Category,
			check.Status,
			check.ImpactScore,
			check.Confidence,
			escapeCell(check.Current),
			escapeCell(check.Expected),
			escapeCell(check.Fix),
		))
	}

	if len(report.Regressions) > 0 {
		b.WriteString("\n## Regressions\n\n")
		b.WriteString("| ID | Previous | Current | Description |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, regression := range report.Regressions {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", regression.ID, regression.Previous, regression.Current, escapeCell(regression.Description)))
		}
	}

	if options.Share {
		b.WriteString("\n")
		b.WriteString(RenderShareSnippet(report))
	}

	return b.String()
}

func escapeCell(value string) string {
	replaced := strings.ReplaceAll(value, "|", "\\|")
	replaced = strings.ReplaceAll(replaced, "\n", "<br>")
	if strings.TrimSpace(replaced) == "" {
		return "-"
	}
	return replaced
}
