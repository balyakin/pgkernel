package policy

import "github.com/ebalyakin/pgkernel/internal/checker"

// FILE:internal/policy/fail.go
// VERSION:1.0.2
// START_MODULE_CONTRACT:
// PURPOSE:Compute final process exit code from check results and policy flags.
// SCOPE:fail-on threshold, runtime errors, and regression-only mode.
// INPUT:Check results, regressions list, runtime error flag.
// OUTPUT:Deterministic exit code in range 0..3.
// KEYWORDS:[DOMAIN(CI): policy gating; CONCEPT(Priority): runtime>crit>warn]
// LINKS:[USES_API(None): pure policy function]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.2 - Restored fail-on threshold behavior in standard mode while preserving runtime and critical priority.
// PREV_CHANGE_SUMMARY:1.0.1 - Aligned standard-mode exit code priority with contract (3>2>1>0).
// END_CHANGE_SUMMARY

func DetermineExitCode(results []checker.CheckResult, regressions []checker.Regression, failOn string, runtimeError bool, regressionsOnly bool) int {
	if runtimeError {
		return 3
	}

	if regressionsOnly {
		return exitCodeFromRegressions(regressions, failOn)
	}

	// BUG_FIX_CONTEXT: Standard mode must still honor --fail-on policy threshold. Previous logic ignored failOn and always failed on warnings.
	hasCrit := false
	hasWarn := false
	for _, result := range results {
		switch result.Status {
		case checker.StatusCrit:
			hasCrit = true
		case checker.StatusWarn:
			hasWarn = true
		}
	}

	if hasCrit {
		return 2
	}

	if normalizeFailOn(failOn) == "warn" {
		if hasWarn {
			return 1
		}
		return 0
	}

	if hasWarn {
		return 0
	}

	return 0
}

func exitCodeFromRegressions(regressions []checker.Regression, failOn string) int {
	if len(regressions) == 0 {
		return 0
	}

	failPolicy := normalizeFailOn(failOn)
	hasCrit := false
	hasWarn := false

	for _, regression := range regressions {
		if regression.Current == checker.StatusCrit {
			hasCrit = true
		}
		if regression.Current == checker.StatusWarn {
			hasWarn = true
		}
	}

	if hasCrit {
		return 2
	}

	if failPolicy == "warn" && hasWarn {
		return 1
	}

	return 0
}

func normalizeFailOn(value string) string {
	if value == "warn" {
		return "warn"
	}
	return "crit"
}
