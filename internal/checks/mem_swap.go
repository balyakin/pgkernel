package checks

import (
	"fmt"

	"github.com/ebalyakin/pgkernel/internal/checker"
)

// FILE:internal/checks/mem_swap.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Implement memory pressure and OOM sensitivity checks.
// SCOPE:MEM-003, MEM-004, MEM-005.
// INPUT:sysctl values and PostgreSQL process metadata.
// OUTPUT:Structured memory risk assessment.
// KEYWORDS:[DOMAIN(Memory): swappiness and overcommit; DOMAIN(Reliability): OOM resilience]
// LINKS:[READS_DATA_FROM(/proc/sys/vm/*): sysctl]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added swap and OOM checks.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

type mem003Swappiness struct{}
type mem004Overcommit struct{}
type mem005OOMAdj struct{}

func MemorySwapChecks() []checker.Check {
	return []checker.Check{
		mem003Swappiness{},
		mem004Overcommit{},
		mem005OOMAdj{},
	}
}

func (c mem003Swappiness) Meta() checker.Meta {
	return checker.Meta{ID: "MEM-003", Name: "vm.swappiness", Category: "memory"}
}

func (c mem003Swappiness) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "swappiness check requires Linux /proc/sys.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "swappiness is host-level in managed profile.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}

	v := state.System.Swappiness
	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("%d", v),
		Expected:      "1-10",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/proc/sys/vm/swappiness"},
			FallbackUsed: false,
		},
		Fix:       "sysctl -w vm.swappiness=1 && echo 'vm.swappiness = 1' >> /etc/sysctl.d/99-postgresql.conf",
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	switch {
	case v <= 10:
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "Swappiness is in recommended range for PostgreSQL."
	case v <= 60:
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 50
		result.Message = fmt.Sprintf("Swappiness is %d. Lower values reduce risk of swapping PostgreSQL memory pages.", v)
	default:
		result.Status = checker.StatusCrit
		result.Severity = checker.SeverityCrit
		result.ImpactScore = 86
		result.Message = fmt.Sprintf("Swappiness is %d. High value can severely degrade PostgreSQL under memory pressure.", v)
	}

	return result
}

func (c mem004Overcommit) Meta() checker.Meta {
	return checker.Meta{ID: "MEM-004", Name: "vm.overcommit_memory", Category: "memory"}
}

func (c mem004Overcommit) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "overcommit check requires Linux /proc/sys.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "overcommit tuning is host-level in managed profile.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}

	v := state.System.OvercommitMemory
	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("%d", v),
		Expected:      "2",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/proc/sys/vm/overcommit_memory"},
			FallbackUsed: false,
		},
		Fix:       "sysctl -w vm.overcommit_memory=2 && echo 'vm.overcommit_memory = 2' >> /etc/sysctl.d/99-postgresql.conf",
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	switch v {
	case 2:
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "Overcommit disabled (mode 2), reducing unexpected OOM risk for PostgreSQL."
	case 0:
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 46
		result.Message = "Heuristic overcommit mode detected (0). PostgreSQL can still be OOM-killed under pressure."
	case 1:
		result.Status = checker.StatusCrit
		result.Severity = checker.SeverityCrit
		result.ImpactScore = 88
		result.Message = "Always-overcommit mode detected (1). This is unsafe for PostgreSQL servers."
	default:
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 20
		result.Confidence = checker.ConfidenceLow
		result.Message = "Unknown overcommit mode value detected."
	}

	return result
}

func (c mem005OOMAdj) Meta() checker.Meta {
	return checker.Meta{ID: "MEM-005", Name: "OOM Killer Score for PostgreSQL", Category: "memory"}
}

func (c mem005OOMAdj) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "OOM score check requires Linux /proc.<pid> interface.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "OOM score configuration is managed by provider policy.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}

	if !state.Postgres.Detected || state.Postgres.MainPID == 0 || !state.Postgres.OOMScoreKnown {
		return checker.CheckResult{
			ID:            meta.ID,
			Name:          meta.Name,
			Category:      meta.Category,
			Severity:      checker.SeverityInfo,
			Status:        checker.StatusSkip,
			Current:       "postgres-pid-not-detected",
			Expected:      "oom_score_adj=-1000",
			ImpactScore:   0,
			Confidence:    checker.ConfidenceMedium,
			Applicability: []string{"baremetal", "vm", "container"},
			Evidence: checker.Evidence{
				Sources:      []string{"postmaster.pid", "pgrep"},
				FallbackUsed: true,
			},
			Message: "PostgreSQL PID or oom_score_adj is unavailable, check skipped.",
			Fix:     "PG_PID=$(head -1 /var/lib/postgresql/*/main/postmaster.pid 2>/dev/null || pgrep -o postgres) && echo -1000 > /proc/$PG_PID/oom_score_adj",
			Remediation: checker.Remediation{
				SafetyLevel:    checker.SafetyRuntime,
				RequiresRoot:   true,
				RequiresReboot: false,
			},
			Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		}
	}

	v := state.Postgres.OOMScoreAdj
	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("%d", v),
		Expected:      "-1000",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{fmt.Sprintf("/proc/%d/oom_score_adj", state.Postgres.MainPID)},
			FallbackUsed: false,
		},
		Fix:       "echo -1000 > /proc/$PG_PID/oom_score_adj",
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	if v == -1000 {
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "OOM killer exclusion is fully enabled for PostgreSQL (oom_score_adj=-1000)."
		return result
	}

	if v < 0 {
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 20
		result.Message = "OOM score is reduced for PostgreSQL, but not fully protected to -1000."
		return result
	}

	result.Status = checker.StatusWarn
	result.Severity = checker.SeverityWarn
	result.ImpactScore = 58
	result.Message = "PostgreSQL process has no OOM protection. It may be killed during memory pressure."
	return result
}
