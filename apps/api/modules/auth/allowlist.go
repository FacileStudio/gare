package auth

import (
	"crypto/subtle"
	"strings"
)

// allowlist is the set of addresses allowed to sign in, normalised once so a
// lookup never depends on how the operator typed an entry.
type allowlist struct {
	emails [][]byte
}

// ParseEmails splits a comma-separated GARE_ADMIN_EMAILS value into trimmed,
// lowercased addresses and drops empty entries, so a trailing comma or a stray
// space never becomes an entry that matches nothing or everything.
func ParseEmails(raw string) []string {
	var emails []string
	for _, entry := range strings.Split(raw, ",") {
		if email := normalizeEmail(entry); email != "" {
			emails = append(emails, email)
		}
	}
	return emails
}

// newAllowlist builds the set from addresses in any case or spacing.
func newAllowlist(emails []string) allowlist {
	list := allowlist{}
	for _, email := range emails {
		if normalized := normalizeEmail(email); normalized != "" {
			list.emails = append(list.emails, []byte(normalized))
		}
	}
	return list
}

// allows reports whether email is on the list. It compares against every
// entry without stopping at a match, so the time it takes says nothing about
// which entry matched or whether one did.
func (a allowlist) allows(email string) bool {
	candidate := []byte(normalizeEmail(email))
	found := 0
	for _, entry := range a.emails {
		found |= subtle.ConstantTimeCompare(entry, candidate)
	}
	return found == 1 && len(candidate) > 0
}

// normalizeEmail is the one spelling both sides of a comparison use: porte
// lowercases the addresses it stores, so the list is lowercased to match.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
