package checker

import (
	"time"

	"github.com/google/uuid"
)

// FILE:internal/checker/result.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Define stable report contracts shared across runner, policy, and output layers.
// SCOPE:Check result schema, report envelope, summary counters, compatibility declarations.
// INPUT:Check evaluation artifacts from checker modules.
// OUTPUT:Machine-readable report model for pretty/json/markdown renderers.
// KEYWORDS:[DOMAIN(Observability): report contract; CONCEPT(Compatibility): schema stability; TECH(Go): typed enums]
// LINKS:[USES_API(JSON): encoding/json; READS_DATA_FROM(Runtime): checker context]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Introduced canonical result and report models for pgkernel v0.1 implementation.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

// START_MODULE_MAP:
// TYPE [Weight 1-10][Entity description in English] =>[entity_name_latin]
// FUNC [9][Build report envelope with deterministic metadata] => [NewReport]
// FUNC [8][Initialize summary counters for report checks] => [BuildSummary]
// END_MODULE_MAP

const (
	SchemaVersion = "1.0.0"
	ToolVersion   = "0.1.0"
)

type Severity string

const (
	SeverityInfo Severity = "info"
	SeverityWarn Severity = "warn"
	SeverityCrit Severity = "crit"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusInfo Status = "info"
	StatusWarn Status = "warn"
	StatusCrit Status = "crit"
	StatusSkip Status = "skip"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

type SafetyLevel string

const (
	SafetyRuntime        SafetyLevel = "safe-runtime"
	SafetyRebootRequired SafetyLevel = "reboot-required"
	SafetyHighRisk       SafetyLevel = "high-risk"
)

type Evidence struct {
	Sources      []string `json:"sources"`
	FallbackUsed bool     `json:"fallback_used"`
}

type Remediation struct {
	SafetyLevel    SafetyLevel `json:"safety_level"`
	RequiresRoot   bool        `json:"requires_root"`
	RequiresReboot bool        `json:"requires_reboot"`
}

type CheckResult struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Category      string      `json:"category"`
	Severity      Severity    `json:"severity"`
	Status        Status      `json:"status"`
	Current       string      `json:"current"`
	Expected      string      `json:"expected"`
	ImpactScore   int         `json:"impact_score"`
	Confidence    Confidence  `json:"confidence"`
	Applicability []string    `json:"applicability"`
	Evidence      Evidence    `json:"evidence"`
	Message       string      `json:"message"`
	Fix           string      `json:"fix"`
	Remediation   Remediation `json:"remediation"`
	Reference     string      `json:"reference"`
}

type SystemInfo struct {
	Kernel   string `json:"kernel"`
	Arch     string `json:"arch"`
	Distro   string `json:"distro"`
	RAMBytes uint64 `json:"ram_bytes"`
	CPUCores int    `json:"cpu_cores"`
	CPUModel string `json:"cpu_model"`
}

type PostgreSQLInfo struct {
	Version     string `json:"version"`
	DataDir     string `json:"data_directory"`
	ConfigFile  string `json:"config_file"`
	ConfigFound bool   `json:"-"`
}

type Summary struct {
	Total     int `json:"total"`
	Passed    int `json:"passed"`
	Info      int `json:"info"`
	Skipped   int `json:"skipped"`
	Warnings  int `json:"warnings"`
	Criticals int `json:"criticals"`
	ExitCode  int `json:"exit_code"`
}

type Compatibility struct {
	BackwardCompatibleWith []string `json:"backward_compatible_with"`
	DeprecationNotice      *string  `json:"deprecation_notice"`
}

type Regression struct {
	ID           string `json:"id"`
	Current      Status `json:"current"`
	Previous     Status `json:"previous"`
	Description  string `json:"description"`
	SeverityBump bool   `json:"severity_bump"`
}

type Report struct {
	SchemaVersion string         `json:"schema_version"`
	Version       string         `json:"version"`
	ReportID      string         `json:"report_id"`
	Timestamp     string         `json:"timestamp"`
	Profile       string         `json:"profile"`
	System        SystemInfo     `json:"system"`
	PostgreSQL    PostgreSQLInfo `json:"postgresql"`
	Checks        []CheckResult  `json:"checks"`
	Summary       Summary        `json:"summary"`
	Compatibility Compatibility  `json:"compatibility"`
	Regressions   []Regression   `json:"regressions,omitempty"`
}

// START_FUNCTION_NewReport
// START_CONTRACT:
// PURPOSE:Construct a complete report envelope with deterministic schema metadata.
// INPUTS:
// - profile runtime profile string => profile: string
// - detected system facts => system: SystemInfo
// - detected postgres facts => pg: PostgreSQLInfo
// - evaluated check rows => checks: []CheckResult
// OUTPUTS:
// - Report - report with generated id and RFC3339 timestamp
// SIDE_EFFECTS: none
// KEYWORDS:[PATTERN(Builder): immutable report envelope; CONCEPT(Contract): stable JSON]
// LINKS:[USES_API(UUID): github.com/google/uuid; USES_API(Time): time.Now]
// COMPLEXITY_SCORE:[3][Linear model assembly]
// END_CONTRACT
func NewReport(profile string, system SystemInfo, pg PostgreSQLInfo, checks []CheckResult) Report {
	return Report{
		SchemaVersion: SchemaVersion,
		Version:       ToolVersion,
		ReportID:      uuid.New().String(),
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Profile:       profile,
		System:        system,
		PostgreSQL:    pg,
		Checks:        checks,
		Summary:       BuildSummary(checks),
		Compatibility: Compatibility{
			BackwardCompatibleWith: []string{"1.0.x"},
			DeprecationNotice:      nil,
		},
	}
}

// START_FUNCTION_BuildSummary
// START_CONTRACT:
// PURPOSE:Aggregate summary counters from check statuses.
// INPUTS:
// - checks to aggregate => checks: []CheckResult
// OUTPUTS:
// - Summary - counts grouped by status classes
// SIDE_EFFECTS: none
// KEYWORDS:[PATTERN(Reduce): status aggregation; CONCEPT(Policy): exit-code groundwork]
// LINKS:[USES_API(None): none]
// COMPLEXITY_SCORE:[4][Single pass over checks]
// END_CONTRACT
func BuildSummary(checks []CheckResult) Summary {
	s := Summary{Total: len(checks)}
	for _, check := range checks {
		switch check.Status {
		case StatusPass:
			s.Passed++
		case StatusInfo:
			s.Info++
		case StatusSkip:
			s.Skipped++
		case StatusWarn:
			s.Warnings++
		case StatusCrit:
			s.Criticals++
		}
	}
	return s
}
