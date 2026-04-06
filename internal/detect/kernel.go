package detect

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/detect/kernel.go
// VERSION:1.0.1
// START_MODULE_CONTRACT:
// PURPOSE:Detect kernel version, architecture, preemption model, and rseq capabilities.
// SCOPE:Linux kernel signals from debugfs, uname output, and boot config fallbacks.
// INPUT:Runtime host filesystem and uname commands.
// OUTPUT:checker.KernelState with confidence-oriented detection metadata.
// KEYWORDS:[DOMAIN(Linux): preemption; CONCEPT(Fallback): layered detection]
// LINKS:[READS_DATA_FROM(/sys/kernel/debug/sched/preempt): preemption; READS_DATA_FROM(/boot/config-*): kernel config]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.1 - Fixed kernel release fallback and removed duplicate boot-config reads.
// PREV_CHANGE_SUMMARY:1.0.0 - Implemented layered kernel and preemption detection with rseq signals.
// END_CHANGE_SUMMARY

// START_FUNCTION_DetectKernelState
// START_CONTRACT:
// PURPOSE:Collect kernel state required by KERN checks with progressive fallback logic.
// INPUTS:
// - none
// OUTPUTS:
// - checker.KernelState - normalized kernel facts
// SIDE_EFFECTS: Reads host files and invokes uname command.
// KEYWORDS:[PATTERN(FallbackChain): debugfs->uname->boot-config; CONCEPT(Resilience): partial data handling]
// LINKS:[USES_API(uname): os/exec; READS_DATA_FROM(debugfs): sched preempt]
// COMPLEXITY_SCORE:[6][Multi-source inference with fallback]
// END_CONTRACT
func DetectKernelState() checker.KernelState {
	state := checker.KernelState{}

	release, _ := runCommand("uname", "-r")
	arch, _ := runCommand("uname", "-m")
	if arch == "" {
		arch = runtime.GOARCH
	}

	state.Version = release
	state.Arch = arch
	state.Major = parseKernelMajor(release)

	bootConfigPath := ""
	bootConfig := ""
	if release != "" {
		bootConfigPath = fmt.Sprintf("/boot/config-%s", release)
		bootConfig, _ = readTrim(bootConfigPath)
	}

	state.PreemptionModel, state.PreemptionSource, state.PreemptionDynamic = detectPreemption(bootConfigPath, bootConfig)
	state.PreemptionUnknown = state.PreemptionModel == ""

	state.RSEQSupported, state.RSEQKnown, state.RSEQSource = detectRSEQ(bootConfigPath, bootConfig)

	return state
}

func detectPreemption(bootConfigPath string, bootConfig string) (string, string, bool) {
	if content, err := readTrim("/sys/kernel/debug/sched/preempt"); err == nil && content != "" {
		selected := parseBracketSelection(content)
		if selected == "" {
			parts := strings.Fields(strings.ToLower(content))
			if len(parts) > 0 {
				selected = strings.Trim(parts[0], "[]")
			}
		}
		selected = normalizePreemption(selected)
		if selected != "" {
			dynamic := strings.Contains(strings.ToLower(content), "dynamic") || strings.Contains(content, "none")
			return selected, "/sys/kernel/debug/sched/preempt", dynamic
		}
	}

	if unameV, err := runCommand("uname", "-v"); err == nil {
		upper := strings.ToUpper(unameV)
		dynamic := strings.Contains(upper, "PREEMPT_DYNAMIC")
		switch {
		case strings.Contains(upper, "PREEMPT_NONE"):
			return "none", "uname -v", dynamic
		case strings.Contains(upper, "PREEMPT_VOLUNTARY"):
			return "voluntary", "uname -v", dynamic
		case strings.Contains(upper, "PREEMPT_LAZY"):
			return "lazy", "uname -v", dynamic
		case strings.Contains(upper, "PREEMPT"):
			return "full", "uname -v", dynamic
		}
	}

	if bootConfig != "" {
		upper := strings.ToUpper(bootConfig)
		dynamic := strings.Contains(upper, "CONFIG_PREEMPT_DYNAMIC=Y")
		switch {
		case strings.Contains(upper, "CONFIG_PREEMPT_NONE=Y"):
			return "none", bootConfigPath, dynamic
		case strings.Contains(upper, "CONFIG_PREEMPT_VOLUNTARY=Y"):
			return "voluntary", bootConfigPath, dynamic
		case strings.Contains(upper, "CONFIG_PREEMPT_LAZY=Y"):
			return "lazy", bootConfigPath, dynamic
		case strings.Contains(upper, "CONFIG_PREEMPT=Y"):
			return "full", bootConfigPath, dynamic
		}
	}

	return "", "", false
}

func detectRSEQ(bootConfigPath string, bootConfig string) (bool, bool, string) {
	if fileExists("/sys/kernel/debug/rseq") {
		return true, true, "/sys/kernel/debug/rseq"
	}
	if value, err := readTrim("/proc/sys/kernel/rseq"); err == nil {
		supported := value == "1" || strings.EqualFold(value, "y") || strings.EqualFold(value, "enabled")
		return supported, true, "/proc/sys/kernel/rseq"
	}

	if bootConfig != "" {
		upper := strings.ToUpper(bootConfig)
		if strings.Contains(upper, "CONFIG_RSEQ=Y") {
			return true, true, bootConfigPath
		}
		if strings.Contains(upper, "CONFIG_RSEQ=N") {
			return false, true, bootConfigPath
		}
	}

	return false, false, ""
}

func normalizePreemption(value string) string {
	s := strings.ToLower(strings.TrimSpace(value))
	s = strings.TrimPrefix(s, "preempt_")
	s = strings.TrimPrefix(s, "config_preempt_")
	s = strings.TrimPrefix(s, "preempt")
	if strings.HasPrefix(s, "_") {
		s = strings.TrimPrefix(s, "_")
	}
	switch s {
	case "none":
		return "none"
	case "voluntary":
		return "voluntary"
	case "lazy":
		return "lazy"
	case "full", "rt", "dynamic":
		return "full"
	default:
		if strings.Contains(s, "none") {
			return "none"
		}
		if strings.Contains(s, "voluntary") {
			return "voluntary"
		}
		if strings.Contains(s, "lazy") {
			return "lazy"
		}
		if strings.Contains(s, "preempt") || strings.Contains(s, "full") {
			return "full"
		}
	}
	return ""
}
