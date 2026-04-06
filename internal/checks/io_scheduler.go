package checks

import (
	"fmt"
	"strings"

	"github.com/ebalyakin/pgkernel/internal/checker"
)

// FILE:internal/checks/io_scheduler.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Implement storage scheduler and dirty page write-back checks.
// SCOPE:IO-001, IO-002.
// INPUT:System storage and vm dirty ratio state.
// OUTPUT:IO-focused recommendations for PostgreSQL reliability.
// KEYWORDS:[DOMAIN(IO): scheduler; DOMAIN(MemoryWriteback): dirty ratios]
// LINKS:[READS_DATA_FROM(/sys/block/*/queue/scheduler): scheduler]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Implemented IO checks.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

type io001Scheduler struct{}
type io002DirtyRatios struct{}

func IOChecks() []checker.Check {
	return []checker.Check{
		io001Scheduler{},
		io002DirtyRatios{},
	}
}

func (c io001Scheduler) Meta() checker.Meta {
	return checker.Meta{ID: "IO-001", Name: "I/O Scheduler", Category: "io"}
}

func (c io001Scheduler) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "I/O scheduler check requires Linux sysfs interfaces.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "I/O scheduler is controlled by provider in managed profile.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}

	device := state.System.BlockDevice
	schedulerRaw := state.System.IOScheduler
	selected := strings.ToLower(parseSelectedValue(schedulerRaw))

	if device == "" || selected == "" {
		return checker.CheckResult{
			ID:            meta.ID,
			Name:          meta.Name,
			Category:      meta.Category,
			Severity:      checker.SeverityInfo,
			Status:        checker.StatusSkip,
			Current:       "undetected",
			Expected:      "none or mq-deadline",
			ImpactScore:   0,
			Confidence:    checker.ConfidenceMedium,
			Applicability: []string{"baremetal", "vm", "container"},
			Evidence: checker.Evidence{
				Sources:      []string{"/proc/self/mountinfo", "/sys/block/*/queue/scheduler"},
				FallbackUsed: true,
			},
			Message: "PostgreSQL data block device or scheduler could not be resolved.",
			Fix:     "Set scheduler manually, for example: echo mq-deadline > /sys/block/sda/queue/scheduler",
			Remediation: checker.Remediation{
				SafetyLevel:    checker.SafetyRuntime,
				RequiresRoot:   true,
				RequiresReboot: false,
			},
			Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		}
	}

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("%s=%s", device, selected),
		Expected:      "none or mq-deadline",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{fmt.Sprintf("/sys/block/%s/queue/scheduler", device)},
			FallbackUsed: false,
		},
		Fix:       fmt.Sprintf("echo mq-deadline > /sys/block/%s/queue/scheduler", device),
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	switch selected {
	case "none", "mq-deadline":
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "I/O scheduler is suitable for PostgreSQL database workloads."
	case "bfq", "cfq", "kyber":
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 48
		result.Message = fmt.Sprintf("Scheduler %s may add latency overhead for database workloads.", selected)
	default:
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 35
		result.Message = fmt.Sprintf("Scheduler %s is not in preferred list for PostgreSQL throughput.", selected)
	}

	return result
}

func (c io002DirtyRatios) Meta() checker.Meta {
	return checker.Meta{ID: "IO-002", Name: "vm.dirty_background_ratio / vm.dirty_ratio", Category: "io"}
}

func (c io002DirtyRatios) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "Dirty ratio check requires Linux /proc/sys.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "Dirty ratio tuning is provider managed in managed profile.", "https://www.postgresql.org/docs/current/kernel-resources.html")
	}

	background := state.System.DirtyBackgroundRatio
	dirty := state.System.DirtyRatio
	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("%d/%d", background, dirty),
		Expected:      "<=5/<=20",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/proc/sys/vm/dirty_background_ratio", "/proc/sys/vm/dirty_ratio"},
			FallbackUsed: false,
		},
		Fix:       "sysctl -w vm.dirty_background_ratio=3 && sysctl -w vm.dirty_ratio=10",
		Reference: "https://www.postgresql.org/docs/current/kernel-resources.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	if background <= 5 && dirty <= 20 {
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "Dirty write-back ratios are tuned for smoother IO flush behavior."
		return result
	}

	result.Status = checker.StatusWarn
	result.Severity = checker.SeverityWarn
	result.ImpactScore = 45
	result.Message = "Dirty ratios are high and may cause bursty write-back stalls for PostgreSQL."
	return result
}
