package storage

import (
	"strconv"
	"strings"
)

// ParseDotEnv parses environment variable contents supporting multiline values and quotes.
func ParseDotEnv(content string) (map[string]string, error) {
	res := make(map[string]string)
	lines := strings.Split(content, "\n")
	var acc string
	for _, line := range lines {
		acc = processDotEnvLine(res, line, acc)
	}
	return res, nil
}

func processDotEnvLine(res map[string]string, line, acc string) string {
	if acc != "" {
		combined := acc + "\n" + line
		if isValueComplete(combined) {
			assignDotEnvEntry(res, combined)
			return ""
		}
		return combined
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	if !isValueComplete(trimmed) {
		return trimmed
	}
	assignDotEnvEntry(res, trimmed)
	return ""
}

func isValueComplete(entry string) bool {
	_, v, ok := strings.Cut(entry, "=")
	if !ok {
		return true
	}
	val := strings.TrimSpace(v)
	if strings.HasPrefix(val, "\"") {
		return hasClosingQuote(val[1:], '"')
	}
	if strings.HasPrefix(val, "'") {
		return hasClosingQuote(val[1:], '\'')
	}
	return true
}

func hasClosingQuote(s string, q byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == q {
			if q == '\'' || i == 0 || s[i-1] != '\\' {
				return true
			}
		}
	}
	return false
}

func assignDotEnvEntry(res map[string]string, entry string) {
	entry = strings.TrimSpace(entry)
	if after, ok := strings.CutPrefix(entry, "export"); ok {
		entry = strings.TrimSpace(after)
	}
	k, v, ok := strings.Cut(entry, "=")
	if !ok {
		return
	}
	k = strings.TrimSpace(k)
	if k == "" {
		return
	}
	res[k] = cleanDotEnvValue(strings.TrimSpace(v))
}

func cleanDotEnvValue(v string) string {
	if strings.HasPrefix(v, "\"") {
		if end := strings.LastIndex(v, "\""); end > 0 {
			quoted := v[:end+1]
			if unquoted, err := strconv.Unquote(quoted); err == nil {
				return unquoted
			}
			return quoted[1 : len(quoted)-1]
		}
	}
	if strings.HasPrefix(v, "'") {
		if end := strings.LastIndex(v, "'"); end > 0 {
			return v[1:end]
		}
	}
	if idx := strings.Index(v, " #"); idx != -1 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}
