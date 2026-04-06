package checker

import (
	"fmt"
	"sort"
)

type Meta struct {
	ID       string
	Name     string
	Category string
}

type Check interface {
	Meta() Meta
	Run(state RuntimeState) CheckResult
}

type Runner struct {
	checks       []Check
	runtimeError bool
}

func NewRunner(checks []Check) *Runner {
	ordered := make([]Check, len(checks))
	copy(ordered, checks)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Meta().ID < ordered[j].Meta().ID
	})
	return &Runner{checks: ordered}
}

func (r *Runner) Run(state RuntimeState) []CheckResult {
	r.runtimeError = false
	results := make([]CheckResult, 0, len(r.checks))
	for _, c := range r.checks {
		meta := c.Meta()
		result, panicked := runCheckSafely(c, state)
		if panicked {
			r.runtimeError = true
			result = runtimeFailureResult(meta, result.Message)
		}
		result = normalizeCheckResult(meta, result)
		results = append(results, result)
	}
	return results
}

func (r *Runner) HasRuntimeError() bool {
	return r.runtimeError
}

func runCheckSafely(c Check, state RuntimeState) (result CheckResult, panicked bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicked = true
			result = CheckResult{Message: fmt.Sprintf("panic recovered: %v", recovered)}
		}
	}()
	return c.Run(state), false
}

func runtimeFailureResult(meta Meta, details string) CheckResult {
	if details == "" {
		details = "checker panic recovered"
	}
	return CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Severity:      SeverityCrit,
		Status:        StatusCrit,
		Current:       "runtime-error",
		Expected:      "no panic during check execution",
		ImpactScore:   100,
		Confidence:    ConfidenceHigh,
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Evidence: Evidence{
			Sources:      []string{"checker.Run panic recovery"},
			FallbackUsed: false,
		},
		Message: details,
		Fix:     "Report issue with --format json output and recovered panic details.",
		Remediation: Remediation{
			SafetyLevel:    SafetyRuntime,
			RequiresRoot:   false,
			RequiresReboot: false,
		},
		Reference: "https://github.com/balyakin/pgkernel/issues",
	}
}

func normalizeCheckResult(meta Meta, result CheckResult) CheckResult {
	if result.ID == "" {
		result.ID = meta.ID
	}
	if result.Name == "" {
		result.Name = meta.Name
	}
	if result.Category == "" {
		result.Category = meta.Category
	}
	if result.Status == "" {
		result.Status = StatusInfo
	}
	if result.Severity == "" {
		result.Severity = SeverityFromStatus(result.Status)
	}
	if result.ImpactScore < 0 {
		result.ImpactScore = 0
	}
	if result.ImpactScore > 100 {
		result.ImpactScore = 100
	}
	if result.Confidence == "" {
		result.Confidence = ConfidenceMedium
	}
	if len(result.Applicability) == 0 {
		result.Applicability = []string{"baremetal", "vm", "container", "managed"}
	}
	if len(result.Evidence.Sources) == 0 {
		result.Evidence.Sources = []string{"checker"}
	}
	if result.Remediation.SafetyLevel == "" {
		result.Remediation.SafetyLevel = SafetyRuntime
	}
	if result.Message == "" {
		result.Message = "No additional details provided by check implementation."
	}
	return result
}

func SeverityFromStatus(status Status) Severity {
	switch status {
	case StatusCrit:
		return SeverityCrit
	case StatusWarn:
		return SeverityWarn
	default:
		return SeverityInfo
	}
}
