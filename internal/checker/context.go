package checker

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
