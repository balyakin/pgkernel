package policy

import "github.com/balyakin/pgkernel/internal/checker"

func DetermineExitCode(results []checker.CheckResult, regressions []checker.Regression, failOn string, runtimeError bool, regressionsOnly bool) int {
	if runtimeError {
		return 3
	}

	if regressionsOnly {
		return exitCodeFromRegressions(regressions, failOn)
	}

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
