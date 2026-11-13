package dotenv

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
)

var bareValue = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=,-]*$`)

// Format renders variables as an env file that Parse and systemd's EnvironmentFile both read back
// unchanged. Keys are sorted so the file diffs cleanly. A value made only of characters neither
// reader treats specially is written bare; any other value is double-quoted with the characters
// systemd unescapes inside double quotes escaped, so spaces, `#`, quotes and newlines survive.
func Format(envs map[string]string) []byte {
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(formatValue(envs[k]))
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func formatValue(value string) string {
	if bareValue.MatchString(value) {
		return value
	}
	escaper := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + escaper.Replace(value) + `"`
}
