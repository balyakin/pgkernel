package output

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/balyakin/pgkernel/internal/checker"
)

func RenderPretty(report checker.Report, options RenderOptions) string {
	checks := filterBySeverity(report.Checks, options.SeverityFilter)
	grouped := groupByCategory(checks)

	b := &strings.Builder{}
	b.WriteString(fmt.Sprintf("pgkernel v%s — PostgreSQL & Linux Kernel Health Check\n", report.Version))
	b.WriteString("══════════════════════════════════════════════════════════\n\n")
	b.WriteString(fmt.Sprintf("System: %s %s | %s\n", report.System.Kernel, report.System.Arch, report.System.Distro))
	b.WriteString(fmt.Sprintf("PostgreSQL: %s | data: %s\n", report.PostgreSQL.Version, fallbackString(report.PostgreSQL.DataDir, "unknown")))
	b.WriteString(fmt.Sprintf("RAM: %s | CPU: %d cores (%s)\n", humanRAM(report.System.RAMBytes), report.System.CPUCores, fallbackString(report.System.CPUModel, "unknown")))
	b.WriteString(fmt.Sprintf("Profile: %s\n\n", report.Profile))

	categoryOrder := []string{"kernel", "memory", "io", "network", "postgresql"}
	for _, category := range categoryOrder {
		items := grouped[category]
		if len(items) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("── %s ───────────────────────────────────────────\n\n", categoryTitle(category)))
		for _, item := range items {
			icon := statusIcon(item.Status)
			line := fmt.Sprintf("%s %s  %s", icon, item.ID, item.Name)
			b.WriteString(colorizeStatus(line, item.Status, options.NoColor))
			b.WriteString("\n")

			switch item.Status {
			case checker.StatusPass:
				b.WriteString(fmt.Sprintf("            Current:  %s\n", fallbackString(item.Current, "ok")))
				b.WriteString("            Status:   ok\n\n")
			default:
				b.WriteString(fmt.Sprintf("            Current:  %s\n", fallbackString(item.Current, "unknown")))
				b.WriteString(fmt.Sprintf("            Expected: %s\n", fallbackString(item.Expected, "n/a")))
				b.WriteString(fmt.Sprintf("            Impact:   %d/100 | Confidence: %s | Safety: %s\n\n", item.ImpactScore, item.Confidence, item.Remediation.SafetyLevel))
				if item.Message != "" {
					b.WriteString(fmt.Sprintf("            %s\n\n", item.Message))
				}
				if item.Fix != "" {
					b.WriteString(fmt.Sprintf("            Fix: %s\n", item.Fix))
				}
				if item.Reference != "" {
					b.WriteString(fmt.Sprintf("            Ref: %s\n", item.Reference))
				}
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("══════════════════════════════════════════════════════════\n")
	b.WriteString(fmt.Sprintf("Summary: %d checks | %d passed | %d warnings | %d critical | %d skipped\n", report.Summary.Total, report.Summary.Passed, report.Summary.Warnings, report.Summary.Criticals, report.Summary.Skipped))
	b.WriteString("══════════════════════════════════════════════════════════\n")
	if len(report.Regressions) > 0 {
		b.WriteString("\nRegressions:\n")
		for _, regression := range report.Regressions {
			b.WriteString(fmt.Sprintf("- %s: %s -> %s (%s)\n", regression.ID, regression.Previous, regression.Current, regression.Description))
		}
	}

	if options.Share {
		b.WriteString("\n")
		b.WriteString(RenderShareSnippet(report))
	}

	return b.String()
}

func groupByCategory(results []checker.CheckResult) map[string][]checker.CheckResult {
	grouped := make(map[string][]checker.CheckResult)
	for _, result := range results {
		category := result.Category
		grouped[category] = append(grouped[category], result)
	}
	for category := range grouped {
		sort.Slice(grouped[category], func(i, j int) bool {
			return grouped[category][i].ID < grouped[category][j].ID
		})
	}
	return grouped
}

func categoryTitle(category string) string {
	switch category {
	case "kernel":
		return "Kernel Preemption"
	case "memory":
		return "Huge Pages and Memory"
	case "io":
		return "I/O and Writeback"
	case "network":
		return "Networking"
	case "postgresql":
		return "PostgreSQL"
	default:
		return upperFirst(category)
	}
}

func upperFirst(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func statusIcon(status checker.Status) string {
	switch status {
	case checker.StatusPass:
		return "✓"
	case checker.StatusWarn:
		return "⚠"
	case checker.StatusCrit:
		return "✗"
	case checker.StatusSkip:
		return "↷"
	default:
		return "ℹ"
	}
}

func colorizeStatus(value string, status checker.Status, noColor bool) string {
	if noColor {
		return value
	}
	const (
		reset  = "\x1b[0m"
		red    = "\x1b[31m"
		yellow = "\x1b[33m"
		green  = "\x1b[32m"
		cyan   = "\x1b[36m"
		blue   = "\x1b[34m"
	)
	switch status {
	case checker.StatusPass:
		return green + value + reset
	case checker.StatusWarn:
		return yellow + value + reset
	case checker.StatusCrit:
		return red + value + reset
	case checker.StatusSkip:
		return blue + value + reset
	default:
		return cyan + value + reset
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func humanRAM(value uint64) string {
	if value == 0 {
		return "unknown"
	}
	gb := float64(value) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.0f GB", gb)
}
