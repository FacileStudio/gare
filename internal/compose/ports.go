package compose

import (
	"strconv"
	"strings"
)

func parsePortSpec(spec string) (int, int) {
	trimmed := strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(strings.TrimSpace(spec)), "/tcp"), "/udp")
	parts := strings.Split(trimmed, ":")
	if len(parts) == 1 {
		port := firstPort(parts[0])
		return port, port
	}
	return firstPort(parts[len(parts)-2]), firstPort(parts[len(parts)-1])
}

func firstPort(value string) int {
	if idx := strings.Index(value, "-"); idx > 0 {
		value = value[:idx]
	}
	return toPort(value)
}

func toPort(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case string:
		port, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0
		}
		return port
	}
	return 0
}
