package checks

import (
	"fmt"
	"math"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

type mem001THP struct{}
type mem002StaticHugePages struct{}

func MemoryHugePageChecks() []checker.Check {
	return []checker.Check{
		mem001THP{},
		mem002StaticHugePages{},
	}
}

func (c mem001THP) Meta() checker.Meta {
	return checker.Meta{ID: "MEM-001", Name: "Transparent Huge Pages (THP)", Category: "memory"}
}

func (c mem001THP) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "THP check requires Linux sysfs interface.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "THP remediation is host-level and not available in managed profile.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}

	selected := strings.ToLower(parseSelectedValue(state.System.THPEnabled))

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       selected,
		Expected:      "never",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/sys/kernel/mm/transparent_hugepage/enabled", "/sys/kernel/mm/transparent_hugepage/defrag"},
			FallbackUsed: false,
		},
		Fix:       "echo never > /sys/kernel/mm/transparent_hugepage/enabled && echo never > /sys/kernel/mm/transparent_hugepage/defrag",
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	switch selected {
	case "never":
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "THP is disabled (never). This is preferred for stable PostgreSQL latency."
	case "always":
		result.Status = checker.StatusCrit
		result.Severity = checker.SeverityCrit
		result.ImpactScore = 91
		result.Message = "THP is enabled globally (always). PostgreSQL latency spikes due to memory compaction are likely."
	case "madvise":
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 52
		result.Message = "THP is set to madvise. Acceptable, but never is usually better for deterministic PostgreSQL latency."
	default:
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 30
		result.Confidence = checker.ConfidenceLow
		result.Message = "Unable to read THP mode."
		result.Current = "unknown"
	}

	return result
}

func (c mem002StaticHugePages) Meta() checker.Meta {
	return checker.Meta{ID: "MEM-002", Name: "Static Huge Pages Configuration", Category: "memory"}
}

func (c mem002StaticHugePages) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "Static huge pages check requires Linux procfs/sysctl.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "HugePages tuning is usually unavailable in managed profile.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if !state.Postgres.Detected {
		return checker.CheckResult{
			ID:            meta.ID,
			Name:          meta.Name,
			Category:      meta.Category,
			Severity:      checker.SeverityInfo,
			Status:        checker.StatusSkip,
			Current:       "postgresql.conf not detected",
			Expected:      "huge_pages setting available",
			ImpactScore:   0,
			Confidence:    checker.ConfidenceMedium,
			Applicability: []string{"baremetal", "vm", "container"},
			Evidence: checker.Evidence{
				Sources:      []string{"postgresql.conf"},
				FallbackUsed: true,
			},
			Message: "PostgreSQL config was not detected, so static huge pages check is skipped.",
			Fix:     "Provide --pg-config to enable PG checks.",
			Remediation: checker.Remediation{
				SafetyLevel:    checker.SafetyRuntime,
				RequiresRoot:   false,
				RequiresReboot: false,
			},
			Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		}
	}

	hugePagesMode := strings.ToLower(strings.TrimSpace(state.Postgres.Settings["huge_pages"]))
	if hugePagesMode == "" {
		hugePagesMode = "try"
	}

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("huge_pages=%s, HugePages_Total=%d, HugePages_Free=%d", hugePagesMode, state.System.HugePagesTotal, state.System.HugePagesFree),
		Expected:      "huge_pages=on|try and HugePages_Total>0",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/proc/meminfo", state.Postgres.ConfigPath},
			FallbackUsed: false,
		},
		Fix:       "echo 4506 > /proc/sys/vm/nr_hugepages && echo 'vm.nr_hugepages = 4506' >> /etc/sysctl.d/99-postgresql.conf",
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyHighRisk,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	if hugePagesMode == "off" {
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 64
		result.Message = "PostgreSQL huge_pages is off. Enabling static huge pages usually improves TLB efficiency and throughput."
		return result
	}

	if state.System.HugePagesTotal == 0 {
		result.Status = checker.StatusCrit
		result.Severity = checker.SeverityCrit
		result.ImpactScore = 85
		result.Message = "PostgreSQL expects huge pages (on/try) but kernel has zero preallocated huge pages."
		return result
	}

	sharedBuffersRaw := state.Postgres.Settings["shared_buffers"]
	sharedBuffersBytes, ok := parsePGBytesWithDefault(sharedBuffersRaw, pgDefault8KBlocks)
	if ok && state.System.HugePageSizeKB > 0 {
		hugePageBytes := int64(state.System.HugePageSizeKB) * 1024
		neededPages := int(math.Ceil(float64(sharedBuffersBytes) / float64(hugePageBytes)))
		if neededPages > state.System.HugePagesTotal {
			result.Status = checker.StatusWarn
			result.Severity = checker.SeverityWarn
			result.ImpactScore = 70
			result.Message = fmt.Sprintf("Configured huge pages (%d) are likely insufficient for shared_buffers (%s). Estimated minimum pages: %d.", state.System.HugePagesTotal, bytesToHuman(sharedBuffersBytes), neededPages)
			return result
		}
	}

	result.Status = checker.StatusPass
	result.Severity = checker.SeverityInfo
	result.ImpactScore = 0
	result.Message = "Static huge pages configuration is consistent with PostgreSQL expectations."
	return result
}
