package tests

import (
	"testing"

	"github.com/balyakin/pgkernel/internal/checker"
	"github.com/balyakin/pgkernel/internal/checks"
)

func TestKERN001LazyProducesWarning(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "baremetal",
		Kernel: checker.KernelState{
			Version:         "7.0.0-generic",
			Major:           7,
			PreemptionModel: "lazy",
		},
	}

	result := runCheckByID(t, checks.KernelChecks(), "KERN-001", state)
	if result.Status != checker.StatusWarn {
		t.Fatalf("expected KERN-001 to warn for lazy preemption, got %s", result.Status)
	}
}

func TestMEM001AlwaysProducesCritical(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "baremetal",
		System: checker.SystemState{
			THPEnabled: "always [always] madvise never",
		},
	}

	result := runCheckByID(t, checks.MemoryHugePageChecks(), "MEM-001", state)
	if result.Status != checker.StatusCrit {
		t.Fatalf("expected MEM-001 to be critical for THP always, got %s", result.Status)
	}
}

func TestPG001DefaultSharedBuffersOnLargeRAMIsCritical(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "baremetal",
		System: checker.SystemState{
			RAMBytes: 16 * 1024 * 1024 * 1024,
		},
		Postgres: checker.PostgresState{
			Detected:   true,
			ConfigPath: "/etc/postgresql/17/main/postgresql.conf",
			Settings: map[string]string{
				"shared_buffers": "128MB",
			},
		},
	}

	result := runCheckByID(t, checks.PostgresChecks(), "PG-001", state)
	if result.Status != checker.StatusCrit {
		t.Fatalf("expected PG-001 to be critical on large RAM with 128MB shared_buffers, got %s", result.Status)
	}
}

func TestPG002WorkMemUsesKilobyteDefaultUnit(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "baremetal",
		System: checker.SystemState{
			RAMBytes: 512 * 1024 * 1024,
		},
		Postgres: checker.PostgresState{
			Detected:   true,
			ConfigPath: "/etc/postgresql/17/main/postgresql.conf",
			Settings: map[string]string{
				"work_mem":        "4096",
				"max_connections": "200",
			},
		},
	}

	result := runCheckByID(t, checks.PostgresChecks(), "PG-002", state)
	if result.Status != checker.StatusWarn {
		t.Fatalf("expected PG-002 warning when work_mem default unit is normalized to kB, got %s", result.Status)
	}
}

func TestMEM002UsesDetectedHugePageSize(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "baremetal",
		System: checker.SystemState{
			HugePagesTotal: 8,
			HugePagesFree:  2,
			HugePageSizeKB: 1048576,
		},
		Postgres: checker.PostgresState{
			Detected:   true,
			ConfigPath: "/etc/postgresql/17/main/postgresql.conf",
			Settings: map[string]string{
				"huge_pages":     "on",
				"shared_buffers": "8GB",
			},
		},
	}

	result := runCheckByID(t, checks.MemoryHugePageChecks(), "MEM-002", state)
	if result.Status != checker.StatusPass {
		t.Fatalf("expected MEM-002 pass when 1GB huge pages satisfy shared_buffers, got %s", result.Status)
	}
}

func TestIO001SkipsInManagedProfile(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "managed",
		System: checker.SystemState{
			BlockDevice: "sda",
			IOScheduler: "bfq [bfq]",
		},
	}

	result := runCheckByID(t, checks.IOChecks(), "IO-001", state)
	if result.Status != checker.StatusSkip {
		t.Fatalf("expected IO-001 skip for managed profile, got %s", result.Status)
	}
}

func TestPG003WarnsWhenCheckpointTargetMissing(t *testing.T) {
	state := checker.RuntimeState{
		OS:      "linux",
		Profile: "baremetal",
		Postgres: checker.PostgresState{
			Detected:   true,
			ConfigPath: "/etc/postgresql/17/main/postgresql.conf",
			Settings: map[string]string{
				"wal_buffers": "-1",
			},
		},
	}

	result := runCheckByID(t, checks.PostgresChecks(), "PG-003", state)
	if result.Status != checker.StatusWarn {
		t.Fatalf("expected PG-003 warn when checkpoint_completion_target is missing, got %s", result.Status)
	}
}

func runCheckByID(t *testing.T, available []checker.Check, id string, state checker.RuntimeState) checker.CheckResult {
	t.Helper()
	for _, c := range available {
		if c.Meta().ID == id {
			return c.Run(state)
		}
	}
	t.Fatalf("check %s not found", id)
	return checker.CheckResult{}
}
