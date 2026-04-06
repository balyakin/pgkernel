package detect

import (
	"runtime"

	"github.com/balyakin/pgkernel/internal/checker"
)

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
