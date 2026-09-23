// Package dotenv parses and writes dotenv-style environment files.
package dotenv

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Parse reads environment variable contents supporting multiline values, quotes and comments.
func Parse(content string) (map[string]string, error) {
	res := make(map[string]string)
	var acc string
	for line := range strings.SplitSeq(content, "\n") {
		acc = processLine(res, line, acc)
	}
	return res, nil
}

// LoadFile reads key-value pairs from a dotenv file.
func LoadFile(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

// ParseAssignments parses KEY=VALUE argument slices into a map.
func ParseAssignments(args []string) (map[string]string, error) {
	res := make(map[string]string, len(args))
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid environment variable format %q: expected KEY=VALUE", arg)
		}
		res[strings.TrimSpace(key)] = value
	}
	return res, nil
}

func processLine(res map[string]string, line, acc string) string {
	if acc != "" {
		combined := acc + "\n" + line
		if isValueComplete(combined) {
			assignEntry(res, combined)
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
	assignEntry(res, trimmed)
	return ""
}

func isValueComplete(entry string) bool {
	_, value, ok := strings.Cut(entry, "=")
	if !ok {
		return true
	}
	val := strings.TrimSpace(value)
	if strings.HasPrefix(val, `"`) {
		return hasClosingQuote(val[1:], '"')
	}
	if strings.HasPrefix(val, "'") {
		return hasClosingQuote(val[1:], '\'')
	}
	return true
}

func hasClosingQuote(s string, quote byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == quote {
			if quote == '\'' || i == 0 || s[i-1] != '\\' {
				return true
			}
		}
	}
	return false
}

func assignEntry(res map[string]string, entry string) {
	entry = strings.TrimSpace(entry)
	if after, ok := strings.CutPrefix(entry, "export"); ok {
		entry = strings.TrimSpace(after)
	}
	key, value, ok := strings.Cut(entry, "=")
	if !ok {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	res[key] = cleanValue(strings.TrimSpace(value))
}

func cleanValue(v string) string {
	if strings.HasPrefix(v, `"`) {
		if end := strings.LastIndex(v, `"`); end > 0 {
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
