package tests

import (
	"encoding/json"
	"testing"

	"github.com/balyakin/pgkernel/internal/checker"
)

func TestReportSchemaFieldsPresent(t *testing.T) {
	report := checker.NewReport(
		"baremetal",
		checker.SystemInfo{Kernel: "7.0.0", Arch: "x86_64", Distro: "Ubuntu", RAMBytes: 1024, CPUCores: 4, CPUModel: "cpu"},
		checker.PostgreSQLInfo{Version: "17.2", DataDir: "/var/lib/postgresql", ConfigFile: "/etc/postgresql.conf", ConfigFound: true},
		[]checker.CheckResult{{ID: "KERN-001", Status: checker.StatusPass}},
	)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal report map: %v", err)
	}

	requiredFields := []string{"schema_version", "version", "report_id", "timestamp", "profile", "system", "postgresql", "checks", "summary", "compatibility"}
	for _, field := range requiredFields {
		if _, exists := payload[field]; !exists {
			t.Fatalf("expected required field %q in schema payload", field)
		}
	}
}
