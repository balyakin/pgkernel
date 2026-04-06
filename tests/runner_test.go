package tests

import (
	"testing"

	"github.com/balyakin/pgkernel/internal/checker"
)

type panicCheck struct{}

func (p panicCheck) Meta() checker.Meta {
	return checker.Meta{ID: "PANIC-001", Name: "panic check", Category: "runtime"}
}

func (p panicCheck) Run(state checker.RuntimeState) checker.CheckResult {
	panic("forced panic for test")
}

func TestRunnerRecoversCheckPanicAndFlagsRuntimeError(t *testing.T) {
	runner := checker.NewRunner([]checker.Check{panicCheck{}})
	results := runner.Run(checker.RuntimeState{OS: "linux"})

	if !runner.HasRuntimeError() {
		t.Fatal("expected runner runtime error flag after panic")
	}
	if len(results) != 1 {
		t.Fatalf("expected single synthetic result, got %d", len(results))
	}
	if results[0].Status != checker.StatusCrit {
		t.Fatalf("expected synthetic runtime failure status crit, got %s", results[0].Status)
	}
	if results[0].Remediation.SafetyLevel == "" {
		t.Fatal("expected safety level to be populated in synthetic runtime failure result")
	}
}
