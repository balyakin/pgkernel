package detect

import (
	"os"
	"strings"
)

// FILE:internal/detect/profile.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Classify execution environment into baremetal/vm/container/managed profiles.
// SCOPE:Lightweight heuristics used by policy and check applicability.
// INPUT:Filesystem markers and environment variables.
// OUTPUT:Profile string expected by report schema.
// KEYWORDS:[DOMAIN(Runtime): environment profile; CONCEPT(Heuristics): best effort]
// LINKS:[READS_DATA_FROM(/.dockerenv): container marker; READS_DATA_FROM(env): managed hints]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added auto profile detection with explicit override support.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

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
