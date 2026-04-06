package checks

import (
	"fmt"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/checks/net_tcp.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Implement networking baseline checks relevant for remote PostgreSQL clients.
// SCOPE:NET-001.
// INPUT:TCP keepalive sysctl values.
// OUTPUT:Informational or pass state with recommended tuning range.
// KEYWORDS:[DOMAIN(Networking): tcp keepalive; CONCEPT(Resilience): stale connection detection]
// LINKS:[READS_DATA_FROM(/proc/sys/net/ipv4/*): keepalive knobs]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added TCP keepalive check.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

type net001TCPKeepalive struct{}

func NetworkChecks() []checker.Check {
	return []checker.Check{net001TCPKeepalive{}}
}

func (c net001TCPKeepalive) Meta() checker.Meta {
	return checker.Meta{ID: "NET-001", Name: "TCP Keepalive", Category: "network"}
}

func (c net001TCPKeepalive) Run(state checker.RuntimeState) checker.CheckResult {
	meta := c.Meta()
	if state.OS != "linux" {
		return linuxSkip(meta, "TCP keepalive check currently targets Linux sysctl sources.", "https://www.postgresql.org/docs/current/runtime-config-connection.html")
	}

	v := state.System.TCPKeepaliveTime
	result := checker.CheckResult{
		ID:            meta.ID,
		Name:          meta.Name,
		Category:      meta.Category,
		Current:       fmt.Sprintf("time=%ds intvl=%ds probes=%d", state.System.TCPKeepaliveTime, state.System.TCPKeepaliveIntvl, state.System.TCPKeepaliveProbes),
		Expected:      "tcp_keepalive_time <= 600",
		Applicability: []string{"baremetal", "vm", "container", "managed"},
		Confidence:    checker.ConfidenceHigh,
		Evidence: checker.Evidence{
			Sources:      []string{"/proc/sys/net/ipv4/tcp_keepalive_time", "/proc/sys/net/ipv4/tcp_keepalive_intvl", "/proc/sys/net/ipv4/tcp_keepalive_probes"},
			FallbackUsed: false,
		},
		Fix:       "sysctl -w net.ipv4.tcp_keepalive_time=600",
		Reference: "https://www.postgresql.org/docs/current/runtime-config-connection.html",
		Remediation: checker.Remediation{
			SafetyLevel:    checker.SafetyRuntime,
			RequiresRoot:   true,
			RequiresReboot: false,
		},
	}

	if v <= 600 && v > 0 {
		result.Status = checker.StatusPass
		result.Severity = checker.SeverityInfo
		result.ImpactScore = 0
		result.Message = "TCP keepalive timeout is in recommended range for faster dead-connection detection."
		return result
	}

	result.Status = checker.StatusInfo
	result.Severity = checker.SeverityInfo
	result.ImpactScore = 18
	result.Message = fmt.Sprintf("tcp_keepalive_time=%ds. For remote PostgreSQL clients, 300-600 seconds is usually preferred.", v)
	return result
}
