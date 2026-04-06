package tests

import (
	"testing"

	"github.com/balyakin/pgkernel/internal/checker"
	"github.com/balyakin/pgkernel/internal/policy"
)

type safeCheck struct{}

func (s safeCheck) Meta() checker.Meta {
	return checker.Meta{ID: "SAFE-001", Name: "safe", Category: "kernel"}
}

func (s safeCheck) Run(state checker.RuntimeState) checker.CheckResult {
	return checker.CheckResult{ID: "SAFE-001", Name: "safe", Category: "kernel", Status: checker.StatusPass}
}

func TestApplyCheckFilterExcludesPanickingCheckBeforeExecution(t *testing.T) {
	available := []checker.Check{panicCheck{}, safeCheck{}}
	filtered := policy.ApplyCheckFilter(available, "", "PANIC-001")

	runner := checker.NewRunner(filtered)
	results := runner.Run(checker.RuntimeState{OS: "linux"})

	if runner.HasRuntimeError() {
		t.Fatal("expected no runtime error when panicking check is excluded")
	}
	if len(results) != 1 {
		t.Fatalf("expected only one executed check after filtering, got %d", len(results))
	}
	if results[0].ID != "SAFE-001" {
		t.Fatalf("expected SAFE-001 to execute, got %s", results[0].ID)
	}
}
