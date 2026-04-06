package policy

import (
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

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
