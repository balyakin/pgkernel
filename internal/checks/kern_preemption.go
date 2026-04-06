package checks

import (
	"fmt"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/checks/kern_preemption.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Implement kernel preemption and kernel-version interaction checks.
// SCOPE:KERN-001, KERN-002, KERN-003.
// INPUT:checker.RuntimeState kernel attributes.
// OUTPUT:Structured check results with remediation metadata.
// KEYWORDS:[DOMAIN(Kernel): preemption model; DOMAIN(PostgreSQL): throughput risk]
// LINKS:[READS_DATA_FROM(internal/detect/kernel.go): KernelState]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Implemented KERN check family according to specification.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

type kern001PreemptionModel struct{}
type kern002KernelVersionRisk struct{}
type kern003RSEQSupport struct{}

func KernelChecks() []checker.Check {
	return []checker.Check{
		kern001PreemptionModel{},
		kern002KernelVersionRisk{},
		kern003RSEQSupport{},
	}
}

func (c kern001PreemptionModel) Meta() checker.Meta {
	return checker.Meta{ID: "KERN-001", Name: "Preemption Model", Category: "kernel"}
}

func (c kern001PreemptionModel) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "Kernel preemption check is available only on Linux hosts.", "https://www.phoronix.com/news/Linux-7.0-AWS-PostgreSQL-Drop")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "Kernel preemption cannot be remediated from managed database runtimes.", "https://www.phoronix.com/news/Linux-7.0-AWS-PostgreSQL-Drop")
	}

	model := state.Kernel.PreemptionModel
	current := model
	if current == "" {
		current = "unknown"
	}

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       current,
		Expected:      "none",
		Confidence:    checker.ConfidenceHigh,
		Applicability: []string{"baremetal", "vm", "container"},
		Evidence: checker.Evidence{
			Sources:      preemptionEvidenceSources(state.Kernel.Version),
			FallbackUsed: state.Kernel.PreemptionSource != "/sys/kernel/debug/sched/preempt",
		},
		Fix:       "echo none > /sys/kernel/debug/sched/preempt",
		Reference: "https://www.phoronix.com/news/Linux-7.0-AWS-PostgreSQL-Drop",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	switch model {
	case "none":
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "PREEMPT_NONE detected. Best throughput-oriented model for PostgreSQL workloads."
	case "voluntary":
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 10
		result.Message = "PREEMPT_VOLUNTARY detected. Acceptable profile for PostgreSQL with moderate scheduling overhead."
	case "lazy":
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 78
		result.Message = "PREEMPT_LAZY detected. Linux 7.0 defaults may reduce PostgreSQL throughput on high-core spinlock-heavy workloads."
	case "full":
		result.Status = checker.StatusCrit
		result.Severity = checker.SeverityCrit
		result.ImpactScore = 92
		result.Message = "Full preemption is not recommended for database servers. Significant throughput degradation is expected."
	default:
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 30
		result.Confidence = checker.ConfidenceLow
		result.Message = "Could not determine preemption model. Mount debugfs or provide readable /boot/config-* to increase confidence."
		result.Fix = "mount -t debugfs debugfs /sys/kernel/debug"
	}

	return result
}

func preemptionEvidenceSources(kernelVersion string) []string {
	sources := []string{"/sys/kernel/debug/sched/preempt", "uname -v"}
	if kernelVersion != "" {
		sources = append(sources, fmt.Sprintf("/boot/config-%s", kernelVersion))
	}
	return sources
}

func (c kern002KernelVersionRisk) Meta() checker.Meta {
	return checker.Meta{ID: "KERN-002", Name: "Kernel Version Warning for 7.0+", Category: "kernel"}
}

func (c kern002KernelVersionRisk) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "Kernel version preemption interaction check is Linux-only.", "https://www.phoronix.com/news/Linux-Restrict-Preempt-Modes")
	}
	if state.Profile == "managed" {
		return managedSkip(meta, "Kernel version tuning is informational in managed profile.", "https://www.phoronix.com/news/Linux-Restrict-Preempt-Modes")
	}

	model := state.Kernel.PreemptionModel
	if model == "" {
		model = "unknown"
	}

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("kernel=%s, preemption=%s", state.Kernel.Version, model),
		Expected:      "kernel<7.0 or preemption=none",
		Applicability: []string{"baremetal", "vm", "container"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"uname -r", "KERN-001 detector"},
			FallbackUsed: false,
		},
		Fix:       "echo none > /sys/kernel/debug/sched/preempt",
		Reference: "https://www.phoronix.com/news/Linux-Restrict-Preempt-Modes",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	if state.Kernel.Major > 0 && state.Kernel.Major < 7 {
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "Kernel release is below 7.0. Linux 7.0 preemption-default regression condition is not active."
		return result
	}

	if state.Kernel.Major >= 7 && state.Kernel.PreemptionModel != "none" {
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 68
		if state.Kernel.PreemptionUnknown {
			result.Confidence = checker.ConfidenceLow
			result.Message = "Kernel is 7.0+ and preemption model is unknown. Throughput may be affected if PREEMPT_NONE is not active."
		} else {
			result.Message = "Kernel is 7.0+ and preemption is not set to none. PostgreSQL throughput can regress under contention."
		}
		return result
	}

	if state.Kernel.Major >= 7 && state.Kernel.PreemptionModel == "none" {
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "Kernel is 7.0+ but PREEMPT_NONE is active. Risk from default preemption change is mitigated."
		return result
	}

	result.Status = checker.StatusInfo
	result.Severity = checker.SeverityInfo
	result.ImpactScore = 25
	result.Confidence = checker.ConfidenceLow
	result.Message = "Could not determine kernel major version reliably."
	return result
}

func (c kern003RSEQSupport) Meta() checker.Meta {
	return checker.Meta{ID: "KERN-003", Name: "RSEQ Support", Category: "kernel"}
}

func (c kern003RSEQSupport) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "RSEQ support check is Linux-only.", "https://lore.kernel.org/lkml/")
	}

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("supported=%t", state.Kernel.RSEQSupported),
		Expected:      "supported=true",
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/sys/kernel/debug/rseq", "/proc/sys/kernel/rseq", state.Kernel.RSEQSource},
			FallbackUsed: state.Kernel.RSEQSource != "/sys/kernel/debug/rseq",
		},
		Fix:       "Use a kernel build with CONFIG_RSEQ=y",
		Reference: "https://lore.kernel.org/lkml/",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRebootRequired,
			RequiresRoot:   true,
			RequiresReboot: true,
		},
	}

	if state.Kernel.RSEQKnown && state.Kernel.RSEQSupported {
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 20
		result.Message = "Kernel reports rseq support. Future PostgreSQL scheduling optimizations can take advantage of this capability."
		return result
	}

	if state.Kernel.Major >= 7 {
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 35
		result.Message = "Kernel 7.0+ without confirmed rseq support. Future mitigations for preemption overhead may be unavailable."
		if !state.Kernel.RSEQKnown {
			result.Confidence = checker.ConfidenceLow
		}
		return result
	}

	result.Status = checker.StatusInfo
	result.Severity = checker.SeverityInfo
	result.ImpactScore = 10
	result.Message = "rseq support not detected; for kernels below 7.0 this is informational."
	if !state.Kernel.RSEQKnown {
		result.Confidence = checker.ConfidenceLow
	}
	return result
}
