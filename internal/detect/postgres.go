package detect

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
)

// FILE:internal/detect/postgres.go
// VERSION:1.0.0
// START_MODULE_CONTRACT:
// PURPOSE:Discover PostgreSQL configuration and process-level state without DB connectivity.
// SCOPE:Config path lookup, postgresql.conf parsing, version/data-dir/PID/OOM metadata.
// INPUT:CLI override, host files, and optional local commands.
// OUTPUT:checker.PostgresState with parsed settings map for PG checks.
// KEYWORDS:[DOMAIN(PostgreSQL): configuration; CONCEPT(Discovery): multi-source fallback]
// LINKS:[READS_DATA_FROM(postgresql.conf): settings; USES_API(pg_lsclusters): discovery fallback]
// END_MODULE_CONTRACT

// START_CHANGE_SUMMARY:
// LAST_CHANGE:1.0.0 - Added postgres discovery pipeline with conservative fallbacks.
// PREV_CHANGE_SUMMARY:none
// END_CHANGE_SUMMARY

func DetectPostgresState(explicitConfigPath string) checker.PostgresState {
	state := checker.PostgresState{
		Settings: make(map[string]string),
	}

	configPath := detectConfigPath(explicitConfigPath)
	if configPath == "" {
		return state
	}

	settings, err := parsePostgresConfig(configPath)
	if err != nil {
		return state
	}

	state.Detected = true
	state.ConfigPath = configPath
	state.Settings = settings
	state.DataDir = detectDataDir(configPath, settings)
	state.Version = detectPostgresVersion()

	state.MainPID = detectMainPID(state.DataDir)
	if state.MainPID > 0 {
		oomAdjPath := fmt.Sprintf("/proc/%d/oom_score_adj", state.MainPID)
		if oomScoreAdj, err := readInt(oomAdjPath); err == nil {
			state.OOMScoreAdj = oomScoreAdj
			state.OOMScoreKnown = true
		}
	}

	return state
}

func detectConfigPath(explicitConfigPath string) string {
	if explicitConfigPath != "" && fileExists(explicitConfigPath) {
		return explicitConfigPath
	}

	if clusterPath := detectViaPgLSClusters(); clusterPath != "" {
		return clusterPath
	}

	if pgConfigPath := detectViaPgConfig(); pgConfigPath != "" {
		return pgConfigPath
	}

	commonPatterns := []string{
		"/etc/postgresql/*/main/postgresql.conf",
		"/var/lib/pgsql/*/data/postgresql.conf",
		"/var/lib/postgresql/*/main/postgresql.conf",
	}
	for _, pattern := range commonPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

func detectViaPgLSClusters() string {
	output, err := runCommand("pg_lsclusters")
	if err != nil || output == "" {
		return ""
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Ver") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		version := fields[0]
		cluster := fields[1]
		candidate := fmt.Sprintf("/etc/postgresql/%s/%s/postgresql.conf", version, cluster)
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func detectViaPgConfig() string {
	sysconfDir, err := runCommand("pg_config", "--sysconfdir")
	if err != nil || sysconfDir == "" {
		return ""
	}
	candidate := filepath.Join(sysconfDir, "postgresql.conf")
	if fileExists(candidate) {
		return candidate
	}
	return ""
}

func parsePostgresConfig(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	settings := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if hash := strings.Index(line, "#"); hash >= 0 {
			line = strings.TrimSpace(line[:hash])
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		settings[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

func detectDataDir(configPath string, settings map[string]string) string {
	if v := settings["data_directory"]; v != "" {
		return strings.Trim(v, `"'`)
	}

	re := regexp.MustCompile(`/etc/postgresql/([^/]+)/([^/]+)/postgresql.conf`)
	if m := re.FindStringSubmatch(configPath); len(m) == 3 {
		candidate := fmt.Sprintf("/var/lib/postgresql/%s/%s", m[1], m[2])
		if fileExists(candidate) {
			return candidate
		}
	}

	if pgData := os.Getenv("PGDATA"); pgData != "" {
		return pgData
	}

	return ""
}

func detectPostgresVersion() string {
	parsers := []struct {
		name string
		args []string
	}{
		{name: "psql", args: []string{"--version"}},
		{name: "postgres", args: []string{"--version"}},
	}

	versionRe := regexp.MustCompile(`(\d+\.\d+)`)
	for _, parser := range parsers {
		output, err := runCommand(parser.name, parser.args...)
		if err != nil {
			continue
		}
		if match := versionRe.FindStringSubmatch(output); len(match) > 1 {
			return match[1]
		}
	}
	return "unknown"
}

func detectMainPID(dataDir string) int {
	if dataDir != "" {
		pidPath := filepath.Join(dataDir, "postmaster.pid")
		if fileExists(pidPath) {
			f, err := os.Open(pidPath)
			if err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				if scanner.Scan() {
					if pid, convErr := strconv.Atoi(strings.TrimSpace(scanner.Text())); convErr == nil {
						return pid
					}
				}
			}
		}
	}

	pidOutput, err := runCommand("pgrep", "-o", "postgres")
	if err != nil || pidOutput == "" {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(pidOutput))
	if err != nil {
		return 0
	}
	return pid
}
