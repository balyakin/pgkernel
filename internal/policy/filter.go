package policy

import (
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/policy/filter.go
// VERSION:1.0.1
// START_MODULE_CONTRACT:
// PURPOSE:Apply ID/category include-exclude filters to check result stream.
// SCOPE:--only and --exclude policy routing.
// INPUT:Check results and raw filter expressions.
// OUTPUT:Filtered result slice preserving original order.
// KEYWORDS:[DOMAIN(Policy): check routing; CONCEPT(Determinism): stable order]
// LINKS:[READS_DATA_FROM(CLI): only/exclude flags]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.1 - Added pre-execution check filtering to ensure excluded checks do not influence runtimeError.
// PREV_CHANGE_SUMMARY:1.0.0 - Added policy filters for id/category selectors.
// END_CHANGE_SUMMARY

func ApplyCheckFilter(items []checker.Check, onlyExpr string, excludeExpr string) []checker.Check {
	if strings.TrimSpace(onlyExpr) == "" && strings.TrimSpace(excludeExpr) == "" {
		return items
	}

	onlySet := buildTokenSet(onlyExpr)
	excludeSet := buildTokenSet(excludeExpr)
	filtered := make([]checker.Check, 0, len(items))

	for _, item := range items {
		meta := item.Meta()
		id := strings.ToLower(meta.ID)
		category := normalizeCategory(strings.ToLower(meta.Category))

		if len(onlySet) > 0 {
			if _, ok := onlySet[id]; !ok {
				if _, okCat := onlySet[category]; !okCat {
					continue
				}
			}
		}

		if _, blockedID := excludeSet[id]; blockedID {
			continue
		}
		if _, blockedCat := excludeSet[category]; blockedCat {
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered
}

func ApplyFilter(results []checker.CheckResult, onlyExpr string, excludeExpr string) []checker.CheckResult {
	if strings.TrimSpace(onlyExpr) == "" && strings.TrimSpace(excludeExpr) == "" {
		return results
	}

	onlySet := buildTokenSet(onlyExpr)
	excludeSet := buildTokenSet(excludeExpr)
	filtered := make([]checker.CheckResult, 0, len(results))

	for _, r := range results {
		id := strings.ToLower(r.ID)
		category := normalizeCategory(strings.ToLower(r.Category))

		if len(onlySet) > 0 {
			if _, ok := onlySet[id]; !ok {
				if _, okCat := onlySet[category]; !okCat {
					continue
				}
			}
		}

		if _, blockedID := excludeSet[id]; blockedID {
			continue
		}
		if _, blockedCat := excludeSet[category]; blockedCat {
			continue
		}

		filtered = append(filtered, r)
	}

	return filtered
}

func buildTokenSet(expr string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range strings.Split(expr, ",") {
		trimmed := strings.ToLower(strings.TrimSpace(token))
		if trimmed == "" {
			continue
		}
		set[normalizeCategory(trimmed)] = struct{}{}
	}
	return set
}

func normalizeCategory(value string) string {
	if value == "pg" || value == "postgres" {
		return "postgresql"
	}
	if value == "mem" {
		return "memory"
	}
	if value == "net" {
		return "network"
	}
	return value
}
