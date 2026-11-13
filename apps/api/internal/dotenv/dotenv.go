// Package dotenv parses and writes dotenv-style environment files.
package dotenv

import (
	"fmt"
	"os"
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
	if strings.HasPrefix(val, `"`) || strings.HasPrefix(val, "'") {
		return closingQuote(val[1:], val[0]) >= 0
	}
	return true
}

func stripExport(entry string) string {
	for _, prefix := range []string{"export ", "export\t"} {
		if after, ok := strings.CutPrefix(entry, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return entry
}

func assignEntry(res map[string]string, entry string) {
	entry = stripExport(strings.TrimSpace(entry))
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
	if strings.HasPrefix(v, `"`) || strings.HasPrefix(v, "'") {
		if end := closingQuote(v[1:], v[0]); end >= 0 {
			return unquoteBody(v[0], v[1:1+end])
		}
	}
	if idx := strings.Index(v, " #"); idx != -1 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}
