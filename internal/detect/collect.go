package detect

import (
	"runtime"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/detect/collect.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Compose detector outputs into a unified RuntimeState and report header structures.
// SCOPE:Orchestration glue between detector modules and checker/output layers.
// INPUT:CLI options for profile and config path.
// OUTPUT:RuntimeState + SystemInfo + PostgreSQLInfo.
// KEYWORDS:[PATTERN(Assembler): state composition; CONCEPT(Separation): detect/check boundary]
// LINKS:[USES_API(internal/detect/*): detector modules]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added collector utility to simplify command orchestration.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

type CollectOptions struct {
	Profile      string
	PGConfigPath string
}

func CollectState(options CollectOptions) (checker.RuntimeState, checker.SystemInfo, checker.PostgreSQLInfo) {
	profile := DetectProfile(options.Profile)
	kernel := DetectKernelState()
	postgres := DetectPostgresState(options.PGConfigPath)
	system := DetectSystemState()
	blockDevice, ioScheduler := DetectStorage(postgres.DataDir)
	system.BlockDevice = blockDevice
	system.IOScheduler = ioScheduler

	runtimeState := checker.RuntimeState{
		OS:       runtime.GOOS,
		Profile:  profile,
		Kernel:   kernel,
		System:   system,
		Postgres: postgres,
	}

	systemInfo := checker.SystemInfo{
		Kernel:   kernel.Version,
		Arch:     kernel.Arch,
		Distro:   system.Distro,
		RAMBytes: system.RAMBytes,
		CPUCores: system.CPUCores,
		CPUModel: system.CPUModel,
	}

	pgInfo := checker.PostgreSQLInfo{
		Version:     postgres.Version,
		DataDir:     postgres.DataDir,
		ConfigFile:  postgres.ConfigPath,
		ConfigFound: postgres.Detected,
	}

	return runtimeState, systemInfo, pgInfo
}
