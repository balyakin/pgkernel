package checker

import (
	"fmt"
	"sort"
)

// FILE:internal/checker/checker.go
// VERSION:1.0.1
// START_MODULE_CONTRACT:
// PURPOSE:Provide check interface and deterministic runner execution order.
// SCOPE:Check metadata contract, runner assembly, status-to-severity mapping.
// INPUT:RuntimeState snapshot and check implementations.
// OUTPUT:Ordered list of CheckResult entries.
// KEYWORDS:[PATTERN(Strategy): pluggable checks; PATTERN(Pipeline): evaluation runner]
// LINKS:[READS_DATA_FROM(internal/checker/context.go): RuntimeState]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.1 - Added panic recovery, runtime failure propagation, and result normalization to preserve contract stability.
// PREV_CHANGE_SUMMARY:1.0.0 - Implemented check strategy interface and deterministic execution.
// END_CHANGE_SUMMARY

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

// START_FUNCTION_NewRunner
// START_CONTRACT:
// PURPOSE:Create a runner that executes checks in deterministic ID order.
// INPUTS:
// - provided checks => checks: []Check
// OUTPUTS:
// - *Runner - ready-to-run checker engine
// SIDE_EFFECTS: none
// KEYWORDS:[PATTERN(Sorting): stable execution; CONCEPT(Reproducibility): deterministic output]
// LINKS:[USES_API(sort.Slice): ordering by check id]
// COMPLEXITY_SCORE:[3][single sort operation]
// END_CONTRACT
func NewRunner(checks []Check) *Runner {
	ordered := make([]Check, len(checks))
	copy(ordered, checks)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Meta().ID < ordered[j].Meta().ID
	})
	return &Runner{checks: ordered}
}

// START_FUNCTION_Run
// START_CONTRACT:
// PURPOSE:Execute all registered checks against runtime state.
// INPUTS:
// - runtime snapshot => state: RuntimeState
// OUTPUTS:
// - []CheckResult - sorted check result rows
// SIDE_EFFECTS: none
// KEYWORDS:[PATTERN(Pipeline): sequential execution; CONCEPT(Stateless): pure evaluation]
// LINKS:[USES_API(None): none]
// COMPLEXITY_SCORE:[3][single loop over checks]
// END_CONTRACT
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

// START_FUNCTION_HasRuntimeError
// START_CONTRACT:
// PURPOSE:Expose whether any checker execution produced runtime failure.
// INPUTS:
// - none
// OUTPUTS:
// - bool - true when at least one checker panic was recovered
// SIDE_EFFECTS: none
// KEYWORDS:[CONCEPT(Reliability): runtime signal propagation]
// LINKS:[USES_API(None): none]
// COMPLEXITY_SCORE:[1][field accessor]
// END_CONTRACT
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
		Reference: "https://github.com/ebalyakin/pgkernel/issues",
	}
}

func normalizeCheckResult(meta Meta, result CheckResult) CheckResult {
	// BUG_FIX_CONTEXT: Earlier checker outputs could miss required contract fields, which broke JSON stability and safety classification. The normalizer guarantees complete machine-readable rows for every check.
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

// START_FUNCTION_SeverityFromStatus
// START_CONTRACT:
// PURPOSE:Normalize output severity from resulting status.
// INPUTS:
// - status emitted by check => status: Status
// OUTPUTS:
// - Severity - mapped urgency level
// SIDE_EFFECTS: none
// KEYWORDS:[PATTERN(Mapping): status normalization]
// LINKS:[USES_API(None): none]
// COMPLEXITY_SCORE:[2][constant switch]
// END_CONTRACT
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
