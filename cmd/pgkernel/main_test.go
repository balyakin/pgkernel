package main

import (
	"testing"

	"github.com/ebalyakin/pgkernel/internal/checker"
)

func TestDedupeRegressions(t *testing.T) {
	input := []checker.Regression{
		{ID: "KERN-001", Previous: checker.StatusPass, Current: checker.StatusWarn},
		{ID: "KERN-001", Previous: checker.StatusPass, Current: checker.StatusWarn},
		{ID: "MEM-001", Previous: checker.StatusWarn, Current: checker.StatusCrit},
	}

	unique := dedupeRegressions(input)
	if len(unique) != 2 {
		t.Fatalf("expected dedupe to keep two unique regressions, got %d", len(unique))
	}
}
