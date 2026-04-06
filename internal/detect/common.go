package detect

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func readTrim(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFirstMatch(path string, prefix string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix)), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("prefix not found")
}

func readKeyValue(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = strings.Trim(value, `"'`)
	}
	return result, nil
}

func readInt(path string) (int, error) {
	content, err := readTrim(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(content)
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func parseBracketSelection(content string) string {
	for _, token := range strings.Fields(content) {
		if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
			return strings.Trim(token, "[]")
		}
	}
	return ""
}

func parseKernelMajor(version string) int {
	if version == "" {
		return 0
	}
	parts := strings.SplitN(version, ".", 2)
	if len(parts) == 0 {
		return 0
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return major
}
