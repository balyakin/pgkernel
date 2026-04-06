package detect

import (
	"os"
	"strings"
)

func DetectProfile(requested string) string {
	if requested != "" && requested != "auto" {
		return requested
	}

	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "container"
	}

	managedEnvKeys := []string{
		"RDS_HOSTNAME",
		"AURORA_DB_CLUSTER_IDENTIFIER",
		"CLOUD_SQL_INSTANCE",
		"AZURE_POSTGRESQL_HOST",
	}
	for _, key := range managedEnvKeys {
		if os.Getenv(key) != "" {
			return "managed"
		}
	}

	if cpuInfo, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lower := strings.ToLower(string(cpuInfo))
		if strings.Contains(lower, "hypervisor") || strings.Contains(lower, "kvm") || strings.Contains(lower, "vmware") {
			return "vm"
		}
	}

	return "baremetal"
}
