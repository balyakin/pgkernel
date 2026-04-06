package tests

import (
	"testing"

	"github.com/ebalyakin/pgkernel/internal/checker"
	"github.com/ebalyakin/pgkernel/internal/policy"
)

func TestDetectRegressions(t *testing.T) {
	previous := checker.Report{
		Checks: []checker.CheckResult{
			{ID: "MEM-001", Status: checker.StatusPass},
			{ID: "PG-001", Status: checker.StatusWarn},
		},
	}
	current := []checker.CheckResult{
		{ID: "MEM-001", Status: checker.StatusWarn},
		{ID: "PG-001", Status: checker.StatusCrit},
		{ID: "NET-001", Status: checker.StatusInfo},
	}

	regressions := policy.DetectRegressions(current, previous)
	if len(regressions) != 2 {
		t.Fatalf("expected 2 regressions, got %d", len(regressions))
	}

	if regressions[0].ID != "MEM-001" {
		t.Fatalf("expected first regression to be MEM-001, got %s", regressions[0].ID)
	}
	if regressions[1].ID != "PG-001" {
		t.Fatalf("expected second regression to be PG-001, got %s", regressions[1].ID)
	}
}

func TestDetectRegressionsIgnoresUnknownCheckIDs(t *testing.T) {
	previous := checker.Report{Checks: []checker.CheckResult{{ID: "MEM-001", Status: checker.StatusPass}}}
	current := []checker.CheckResult{
		{ID: "MEM-001", Status: checker.StatusWarn},
		{ID: "NEW-999", Status: checker.StatusCrit},
	}

	regressions := policy.DetectRegressions(current, previous)
	if len(regressions) != 1 {
		t.Fatalf("expected only known-check regression, got %d", len(regressions))
	}
	if regressions[0].ID != "MEM-001" {
		t.Fatalf("expected MEM-001 regression, got %s", regressions[0].ID)
	}
}

func TestSeverityBumpSemantics(t *testing.T) {
	previous := checker.Report{
		Checks: []checker.CheckResult{
			{ID: "A", Status: checker.StatusPass},
			{ID: "B", Status: checker.StatusWarn},
			{ID: "C", Status: checker.StatusPass},
		},
	}
	current := []checker.CheckResult{
		{ID: "A", Status: checker.StatusWarn},
		{ID: "B", Status: checker.StatusCrit},
		{ID: "C", Status: checker.StatusInfo},
	}

	regressions := policy.DetectRegressions(current, previous)
	if len(regressions) != 3 {
		t.Fatalf("expected 3 regressions (pass->warn, warn->crit, pass->info), got %d", len(regressions))
	}

	byID := map[string]checker.Regression{}
	for _, r := range regressions {
		byID[r.ID] = r
	}
	if byID["A"].SeverityBump {
		t.Fatal("expected pass->warn not to be marked as severity bump")
	}
	if !byID["B"].SeverityBump {
		t.Fatal("expected warn->crit to be marked as severity bump")
	}
	if byID["C"].SeverityBump {
		t.Fatal("expected pass->info not to be marked as severity bump")
	}
}
