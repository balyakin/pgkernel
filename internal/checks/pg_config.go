package checks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

type pg001SharedBuffers struct{}
type pg002WorkMem struct{}
type pg003WAL struct{}

func PostgresChecks() []checker.Check {
	return []checker.Check{
		pg001SharedBuffers{},
		pg002WorkMem{},
		pg003WAL{},
	}
}

func (c pg001SharedBuffers) Meta() checker.Meta {
	return checker.Meta{ID: "PG-001", Name: "shared_buffers vs Total RAM", Category: "postgresql"}
}

func (c pg001SharedBuffers) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if !state.Postgres.Detected {
		return pgSkip(meta)
	}

	raw := state.Postgres.Settings["shared_buffers"]
	valueBytes, ok := parsePGBytesWithDefault(raw, pgDefault8KBlocks)
	if !ok || valueBytes <= 0 || state.System.RAMBytes == 0 {
		return checker.CheckResult{
			ID:            meta.ID,
			Name:          meta.Name,
			Category:      meta.Category,
			Severity:      checker.SeverityInfo,
			Status:        checker.StatusInfo,
			Current:       fmt.Sprintf("shared_buffers=%s", raw),
			Expected:      "20%-35% of RAM",
			ImpactScore:   20,
			Confidence:    checker.ConfidenceLow,
			Applicability: []string{"baremetal", "vm", "container", "managed"},
			Evidence: checker.Evidence{
				Sources:      []string{state.Postgres.ConfigPath, "/proc/meminfo"},
				FallbackUsed: true,
			},
			Message:     "Unable to parse shared_buffers or RAM size for percentage-based validation.",
			Fix:         "Set shared_buffers to approximately 25% of total RAM as a starting point.",
			Remediation: checker.Remediation{SafetyLevel: checker.SafetyRuntime, RequiresRoot: false, RequiresReboot: true},
			Reference:   "https://www.postgresql.org/docs/current/runtime-config-resource.html",
		}
	}

	pct := pctOfRAM(valueBytes, state.System.RAMBytes)
	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("%s (%.1f%% of RAM)", bytesToHuman(valueBytes), pct),
		Expected:      "20%-35% of RAM",
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{state.Postgres.ConfigPath, "/proc/meminfo"},
			FallbackUsed: false,
		},
		Fix:       "Increase shared_buffers to ~25% RAM (unless workload profiling recommends otherwise)",
		Reference: "https://www.postgresql.org/docs/current/runtime-config-resource.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRebootRequired,
			RequiresRoot:   false,
			RequiresReboot: true,
		},
	}

	if valueBytes == 128*1024*1024 && state.System.RAMBytes > 4*1024*1024*1024 {
		result.Status = checker.StatusCrit
		result.Severity = checker.SeverityCrit
		result.ImpactScore = 90
		result.Message = fmt.Sprintf("shared_buffers is default 128MB on a %.0fGB host, which is almost certainly under-tuned.", float64(state.System.RAMBytes)/1024/1024/1024)
		return result
	}

	switch {
	case pct >= 20 && pct <= 35:
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "shared_buffers is within the recommended range."
	case pct < 20:
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 55
		result.Message = fmt.Sprintf("shared_buffers is %.1f%% of RAM. Increasing toward ~25%% may improve cache hit ratio.", pct)
	case pct > 40:
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 45
		result.Message = fmt.Sprintf("shared_buffers is %.1f%% of RAM. High values can increase double-buffering pressure with OS cache.", pct)
	default:
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 10
		result.Message = "shared_buffers is outside ideal band but not obviously unsafe."
	}

	return result
}

func (c pg002WorkMem) Meta() checker.Meta {
	return checker.Meta{ID: "PG-002", Name: "work_mem Sanity", Category: "postgresql"}
}

func (c pg002WorkMem) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if !state.Postgres.Detected {
		return pgSkip(meta)
	}

	workMemRaw := state.Postgres.Settings["work_mem"]
	workMemBytes, workOK := parsePGBytesWithDefault(workMemRaw, pgDefaultKilobytes)
	maxConn := parseIntDefault(state.Postgres.Settings["max_connections"], 100)

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("work_mem=%s, max_connections=%d", workMemRaw, maxConn),
		Expected:      "work_mem * max_connections <= 50% RAM",
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{state.Postgres.ConfigPath, "/proc/meminfo"},
			FallbackUsed: false,
		},
		Fix:       "Reduce work_mem or max_connections to preserve headroom under concurrent sort/hash workloads.",
		Reference: "https://www.postgresql.org/docs/current/runtime-config-resource.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRebootRequired,
			RequiresRoot:   false,
			RequiresReboot: true,
		},
	}

	if !workOK || workMemBytes <= 0 || state.System.RAMBytes == 0 {
		result.Status = checker.StatusInfo
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 15
		result.Confidence = checker.ConfidenceLow
		result.Message = "Unable to compute work_mem headroom due to missing or unparsable values."
		return result
	}

	potential := workMemBytes * int64(maxConn)
	pct := pctOfRAM(potential, state.System.RAMBytes)
	result.Current = fmt.Sprintf("work_mem=%s, max_connections=%d, potential=%.1f%% RAM", bytesToHuman(workMemBytes), maxConn, pct)

	if pct > 50 {
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 60
		result.Message = fmt.Sprintf("Worst-case work_mem allocation reaches %.1f%% of RAM, creating OOM risk under concurrent heavy queries.", pct)
		return result
	}

	result.Status = checker.StatusPass
	result.Severity = checker.SeverityInfo
	result.ImpactScore = 0
	result.Message = "work_mem headroom is within a conservative safety envelope."
	return result
}

func (c pg003WAL) Meta() checker.Meta {
	return checker.Meta{ID: "PG-003", Name: "WAL Configuration", Category: "postgresql"}
}

func (c pg003WAL) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if !state.Postgres.Detected {
		return pgSkip(meta)
	}

	walBuffersRaw := strings.TrimSpace(state.Postgres.Settings["wal_buffers"])
	if walBuffersRaw == "" {
		walBuffersRaw = "-1"
	}
	maxWalSizeRaw := strings.TrimSpace(state.Postgres.Settings["max_wal_size"])
	checkpointTargetRaw := strings.TrimSpace(state.Postgres.Settings["checkpoint_completion_target"])

	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("wal_buffers=%s, max_wal_size=%s, checkpoint_completion_target=%s", walBuffersRaw, maxWalSizeRaw, checkpointTargetRaw),
		Expected:      "wal_buffers=-1 or >=16MB and checkpoint_completion_target>=0.7",
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{state.Postgres.ConfigPath},
			FallbackUsed: false,
		},
		Fix:       "Set wal_buffers=-1 and checkpoint_completion_target=0.9 for balanced WAL flush behavior.",
		Reference: "https://www.postgresql.org/docs/current/wal-configuration.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRebootRequired,
			RequiresRoot:   false,
			RequiresReboot: true,
		},
	}

	warnings := make([]string, 0)

	if walBuffersRaw != "-1" {
		walBytes, ok := parsePGBytesWithDefault(walBuffersRaw, pgDefault8KBlocks)
		if !ok {
			warnings = append(warnings, "wal_buffers value could not be parsed")
			result.Confidence = checker.ConfidenceMedium
		} else if walBytes < 16*1024*1024 {
			warnings = append(warnings, "wal_buffers is below 16MB")
		}
	}

	if checkpointTargetRaw == "" {
		warnings = append(warnings, "checkpoint_completion_target is not explicitly set")
		result.Confidence = checker.ConfidenceMedium
	} else {
		target, err := strconv.ParseFloat(strings.Trim(checkpointTargetRaw, `"'`), 64)
		if err != nil {
			warnings = append(warnings, "checkpoint_completion_target could not be parsed")
			result.Confidence = checker.ConfidenceMedium
		} else if target < 0.7 {
			warnings = append(warnings, "checkpoint_completion_target is below 0.7")
		}
	}

	if len(warnings) > 0 {
		result.Status = checker.StatusWarn
		result.Severity = checker.SeverityWarn
		result.ImpactScore = 45
		result.Message = "WAL configuration has potential throughput risks: " + strings.Join(warnings, "; ") + "."
		return result
	}

	result.Status = checker.StatusPass
	result.Severity = checker.SeverityInfo
	result.ImpactScore = 0
	result.Message = "WAL configuration is within recommended baseline values."
	return result
}

func pgSkip(meta checker.Meta) checker.CheckResult {
	return checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Severity:      checker.SeverityInfo,
		Status:        checker.StatusSkip,
		Current:       "postgresql.conf not detected",
		Expected:      "postgresql.conf available",
		ImpactScore:   0,
		Confidence:    checker.ConfidenceMedium,
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Evidence:      checker.Evidence{Sources: []string{"--pg-config autodetect"}, FallbackUsed: true},
		Message:       "PostgreSQL configuration file not found. PG checks were skipped.",
		Fix:           "Provide --pg-config /path/to/postgresql.conf",
		Remediation:   checker.Remediation{SafetyLevel: checker.SafetyRuntime, RequiresRoot: false, RequiresReboot: false},
		Reference:     "https://www.postgresql.org/docs/current/",
	}
}
