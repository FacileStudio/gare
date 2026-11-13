package dotenv

import (
	"strings"
)

const doubleQuoteEscapable = "\"\\`$"

// closingQuote returns the index in s of the quote that closes a value opened with quote, or -1
// when the value is not closed yet. Inside double quotes a backslash escapes the next byte, as
// systemd's EnvironmentFile parser treats it, so `\"` never closes the value and `\\"` does.
func closingQuote(s string, quote byte) int {
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == quote:
			return i
		case quote == '"' && s[i] == '\\':
			i++
		}
	}
	return -1
}

// unescapeDouble decodes the body of a double-quoted value the way systemd does: a backslash
// before one of " \ ` $ yields that byte, a backslash before a newline joins the lines, and any
// other backslash is kept, so the value gare reads is the value the service receives.
func unescapeDouble(body string) string {
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' || i+1 == len(body) {
			b.WriteByte(body[i])
			continue
		}
		next := body[i+1]
		i++
		if next == '\n' {
			continue
		}
		if !strings.ContainsRune(doubleQuoteEscapable, rune(next)) {
			b.WriteByte('\\')
		}
		b.WriteByte(next)
	}
	return b.String()
}

// unquoteBody decodes a quoted value's body: double quotes honour systemd's escapes, single quotes
// are literal.
func unquoteBody(quote byte, body string) string {
	if quote == '"' {
		return unescapeDouble(body)
	}
	return body
}
