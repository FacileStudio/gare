package podman

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseExposedPort reads a Containerfile or Dockerfile and returns the first exposed port, or 0.
func ParseExposedPort(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if p := parseExposeLine(scanner.Text()); p > 0 {
			return p
		}
	}
	if err := scanner.Err(); err != nil {
		return 0
	}
	return 0
}

func parseExposeLine(line string) int {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return 0
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 2 || !strings.EqualFold(fields[0], "EXPOSE") {
		return 0
	}
	raw, _, _ := strings.Cut(fields[1], "/")
	p, err := strconv.Atoi(raw)
	if err != nil || p <= 0 || p > 65535 {
		return 0
	}
	return p
}

// DetectExposedPort inspects the containerfile in repoDir and returns its first exposed port, or 0.
func DetectExposedPort(repoDir, containerfile string) int {
	cf := containerfile
	if cf == "" {
		detected, err := DetectContainerfile(repoDir)
		if err != nil {
			return 0
		}
		cf = detected
	}
	return ParseExposedPort(filepath.Join(repoDir, cf))
}
