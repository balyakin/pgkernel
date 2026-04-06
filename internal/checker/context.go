package checker

// FILE:internal/checker/context.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Define normalized runtime state consumed by all check modules.
// SCOPE:Kernel, system, and PostgreSQL signals collected by detect layer.
// INPUT:System and configuration probes from detect package.
// OUTPUT:Stable in-memory state for deterministic check evaluation and tests.
// KEYWORDS:[DOMAIN(Diagnostics): runtime snapshot; CONCEPT(Testability): dependency isolation]
// LINKS:[READS_DATA_FROM(detect): detector outputs]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added strongly typed runtime state model for all checks.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

type KernelState struct {
	Version           string
	Major             int
	Arch              string
	PreemptionModel   string
	PreemptionSource  string
	PreemptionUnknown bool
	PreemptionDynamic bool
	RSEQSupported     bool
	RSEQKnown         bool
	RSEQSource        string
}

type SystemState struct {
	Distro               string
	RAMBytes             uint64
	CPUCores             int
	CPUModel             string
	THPEnabled           string
	THPDefrag            string
	HugePagesTotal       int
	HugePagesFree        int
	HugePageSizeKB       int
	Swappiness           int
	OvercommitMemory     int
	DirtyBackgroundRatio int
	DirtyRatio           int
	TCPKeepaliveTime     int
	TCPKeepaliveIntvl    int
	TCPKeepaliveProbes   int
	BlockDevice          string
	IOScheduler          string
}

type PostgresState struct {
	Detected      bool
	Version       string
	ConfigPath    string
	DataDir       string
	MainPID       int
	OOMScoreAdj   int
	OOMScoreKnown bool
	Settings      map[string]string
}

type RuntimeState struct {
	OS       string
	Profile  string
	Kernel   KernelState
	System   SystemState
	Postgres PostgresState
}
