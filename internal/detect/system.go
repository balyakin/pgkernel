package detect

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

func DetectSystemState() checker.SystemState {
	state := checker.SystemState{}

	state.Distro = detectDistro()
	state.CPUCores = runtime.NumCPU()
	state.CPUModel = detectCPUModel()

	memInfo, _ := parseMemInfo("/proc/meminfo")
	if kb, ok := memInfo["MemTotal"]; ok {
		state.RAMBytes = uint64(kb) * 1024
	}
	state.HugePagesTotal = memInfo["HugePages_Total"]
	state.HugePagesFree = memInfo["HugePages_Free"]
	state.HugePageSizeKB = memInfo["Hugepagesize"]

	state.THPEnabled, _ = readTrim("/sys/kernel/mm/transparent_hugepage/enabled")
	state.THPDefrag, _ = readTrim("/sys/kernel/mm/transparent_hugepage/defrag")

	state.Swappiness, _ = readInt("/proc/sys/vm/swappiness")
	state.OvercommitMemory, _ = readInt("/proc/sys/vm/overcommit_memory")
	state.DirtyBackgroundRatio, _ = readInt("/proc/sys/vm/dirty_background_ratio")
	state.DirtyRatio, _ = readInt("/proc/sys/vm/dirty_ratio")

	state.TCPKeepaliveTime, _ = readInt("/proc/sys/net/ipv4/tcp_keepalive_time")
	state.TCPKeepaliveIntvl, _ = readInt("/proc/sys/net/ipv4/tcp_keepalive_intvl")
	state.TCPKeepaliveProbes, _ = readInt("/proc/sys/net/ipv4/tcp_keepalive_probes")

	return state
}

func DetectStorage(dataDir string) (string, string) {
	if dataDir == "" {
		return "", ""
	}
	device := resolveBlockDeviceFromMount(dataDir)
	if device == "" {
		return "", ""
	}
	schedulerPath := fmt.Sprintf("/sys/block/%s/queue/scheduler", device)
	scheduler, _ := readTrim(schedulerPath)
	return device, scheduler
}

func detectDistro() string {
	if kv, err := readKeyValue("/etc/os-release"); err == nil {
		if pretty := kv["PRETTY_NAME"]; pretty != "" {
			return pretty
		}
		if name := kv["NAME"]; name != "" {
			if version := kv["VERSION"]; version != "" {
				return name + " " + version
			}
			return name
		}
	}
	return runtime.GOOS
}

func detectCPUModel() string {
	value, err := readFirstMatch("/proc/cpuinfo", "model name")
	if err != nil {
		return "unknown"
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(value)
}

func parseMemInfo(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		raw := strings.TrimSpace(parts[1])
		fields := strings.Fields(raw)
		if len(fields) == 0 {
			continue
		}
		value, convErr := strconv.Atoi(fields[0])
		if convErr != nil {
			continue
		}
		result[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func resolveBlockDeviceFromMount(targetPath string) string {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		absPath = targetPath
	}

	content, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return ""
	}

	bestMount := ""
	bestSource := ""
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, " - ")
		if len(parts) != 2 {
			continue
		}
		left := strings.Fields(parts[0])
		right := strings.Fields(parts[1])
		if len(left) < 5 || len(right) < 2 {
			continue
		}
		mountPoint := left[4]
		source := right[1]
		if strings.HasPrefix(absPath, mountPoint) && len(mountPoint) > len(bestMount) {
			bestMount = mountPoint
			bestSource = source
		}
	}

	if bestSource == "" {
		return ""
	}
	bestSource = strings.TrimPrefix(bestSource, "/dev/")
	bestSource = strings.TrimPrefix(bestSource, "mapper/")

	if strings.HasPrefix(bestSource, "nvme") {
		idx := strings.Index(bestSource, "p")
		if idx > 0 {
			return bestSource[:idx]
		}
		return bestSource
	}

	trimmed := strings.TrimRightFunc(bestSource, func(r rune) bool {
		return r >= '0' && r <= '9'
	})
	if trimmed == "" {
		return bestSource
	}
	return trimmed
}
