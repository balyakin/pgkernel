package tests

import (
	"testing"

	"github.com/balyakin/pgkernel/internal/checker"
	"github.com/balyakin/pgkernel/internal/policy"
)

func TestDetermineExitCodePriority(t *testing.T) {
	results := []checker.CheckResult{{ID: "A", Status: checker.StatusWarn}}
	if got := policy.DetermineExitCode(results, nil, "warn", true, false); got != 3 {
		t.Fatalf("expected runtime error priority exit 3, got %d", got)
	}

	results = []checker.CheckResult{{ID: "A", Status: checker.StatusCrit}, {ID: "B", Status: checker.StatusWarn}}
	if got := policy.DetermineExitCode(results, nil, "warn", false, false); got != 2 {
		t.Fatalf("expected critical exit 2, got %d", got)
	}
}

func TestDetermineExitCodeFailOnThreshold(t *testing.T) {
	warningsOnly := []checker.CheckResult{{ID: "A", Status: checker.StatusWarn}}

	if got := policy.DetermineExitCode(warningsOnly, nil, "warn", false, false); got != 1 {
		t.Fatalf("expected fail-on=warn to return 1 for warnings, got %d", got)
	}

	if got := policy.DetermineExitCode(warningsOnly, nil, "crit", false, false); got != 0 {
		t.Fatalf("expected fail-on=crit to ignore warnings and return 0, got %d", got)
	}
}

func TestDetermineExitCodeRegressionMode(t *testing.T) {
	regressions := []checker.Regression{{ID: "MEM-001", Previous: checker.StatusPass, Current: checker.StatusWarn}}
	if got := policy.DetermineExitCode(nil, regressions, "warn", false, true); got != 1 {
		t.Fatalf("expected regression warning to fail with code 1 under fail-on=warn, got %d", got)
	}

	if got := policy.DetermineExitCode(nil, regressions, "crit", false, true); got != 0 {
		t.Fatalf("expected warning regression ignored with fail-on=crit, got %d", got)
	}
}

func TestApplyFilterByCategoryAndID(t *testing.T) {
	results := []checker.CheckResult{
		{ID: "KERN-001", Category: "kernel"},
		{ID: "MEM-001", Category: "memory"},
		{ID: "PG-001", Category: "postgresql"},
	}

	filtered := policy.ApplyFilter(results, "memory,pg-001", "MEM-001")
	if len(filtered) != 1 {
		t.Fatalf("expected one result after filters, got %d", len(filtered))
	}
	if filtered[0].ID != "PG-001" {
		t.Fatalf("expected PG-001 to remain, got %s", filtered[0].ID)
	}
}
