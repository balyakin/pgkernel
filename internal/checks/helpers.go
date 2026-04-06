package checks

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

func linuxSkip(meta checker.Meta, message string, reference string) checker.CheckResult {
	return checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Severity:      checker.SeverityInfo,
		Status:        checker.StatusSkip,
		Current:       "unsupported-os",
		Expected:      "linux",
		ImpactScore:   0,
		Confidence:    checker.ConfidenceHigh,
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Evidence: checker.Evidence{
			Sources:      []string{"runtime.GOOS"},
			FallbackUsed: false,
		},
		Message: message,
		Fix:     "",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   false,
			RequiresReboot: false,
		},
		Reference: reference,
	}
}

func managedSkip(meta checker.Meta, message string, reference string) checker.CheckResult {
	return checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Severity:      checker.SeverityInfo,
		Status:        checker.StatusSkip,
		Current:       "managed-environment",
		Expected:      "host-level access",
		ImpactScore:   0,
		Confidence:    checker.ConfidenceMedium,
		Applicability: []string{"managed"},
		Evidence: checker.Evidence{
			Sources:      []string{"profile:auto"},
			FallbackUsed: true,
		},
		Message: message,
		Fix:     "Informational only in managed profile.",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   false,
			RequiresReboot: false,
		},
		Reference: reference,
	}
}

func parseSelectedValue(raw string) string {
	for _, token := range strings.Fields(raw) {
		if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
			return strings.Trim(token, "[]")
		}
	}
	if raw == "" {
		return ""
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}
	return strings.Trim(parts[0], "[]")
}

const (
	pgDefaultBytes     int64 = 1
	pgDefaultKilobytes int64 = 1024
	pgDefault8KBlocks  int64 = 8 * 1024
)

func parsePGBytes(raw string) (int64, bool) {
	return parsePGBytesWithDefault(raw, pgDefaultBytes)
}

func parsePGBytesWithDefault(raw string, defaultUnitBytes int64) (int64, bool) {
	value := strings.TrimSpace(strings.ToLower(raw))
	value = strings.Trim(value, `"'`)
	if value == "" {
		return 0, false
	}
	value = strings.ReplaceAll(value, " ", "")

	if value == "-1" {
		return -1, true
	}

	if strings.HasSuffix(value, "ms") {
		return 0, false
	}

	type unitScale struct {
		suffix string
		scale  int64
	}
	orderedUnits := []unitScale{
		{suffix: "tb", scale: 1024 * 1024 * 1024 * 1024},
		{suffix: "gb", scale: 1024 * 1024 * 1024},
		{suffix: "mb", scale: 1024 * 1024},
		{suffix: "kb", scale: 1024},
		{suffix: "t", scale: 1024 * 1024 * 1024 * 1024},
		{suffix: "g", scale: 1024 * 1024 * 1024},
		{suffix: "m", scale: 1024 * 1024},
		{suffix: "k", scale: 1024},
		{suffix: "b", scale: 1},
	}

	for _, unit := range orderedUnits {
		if strings.HasSuffix(value, unit.suffix) {
			numPart := strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			number, err := strconv.ParseFloat(numPart, 64)
			if err != nil {
				return 0, false
			}
			return int64(math.Round(number * float64(unit.scale))), true
		}
	}

	if defaultUnitBytes <= 0 {
		defaultUnitBytes = pgDefaultBytes
	}
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return int64(math.Round(num * float64(defaultUnitBytes))), true
}

func parseIntDefault(raw string, fallback int) int {
	v := strings.TrimSpace(raw)
	v = strings.Trim(v, `"'`)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func bytesToHuman(bytes int64) string {
	if bytes <= 0 {
		return "0B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	return fmt.Sprintf("%.0f%s", math.Round(value), units[idx])
}

func pctOfRAM(bytes int64, ramBytes uint64) float64 {
	if bytes <= 0 || ramBytes == 0 {
		return 0
	}
	return (float64(bytes) / float64(ramBytes)) * 100
}
