package env

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	troncerrors "github.com/FacileStudio/tronc/errors"
)

// keyPattern is the shape of a variable name every shell and env file agrees on.
var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Masking: a value shows its first and last revealedEnds characters only from minRevealLength on,
// so a short secret shows nothing at all, and the middle is always the same run of bullets, so the
// mask never tells a value's length.
const (
	minRevealLength = 12
	revealedEnds    = 2
	maskBullets     = 8
)

// mask hides a value for the list.
func mask(value string) string {
	bullets := strings.Repeat("•", maskBullets)
	if utf8.RuneCountInString(value) < minRevealLength {
		return bullets
	}
	runes := []rune(value)
	return string(runes[:revealedEnds]) + bullets + string(runes[len(runes)-revealedEnds:])
}

// entries masks every variable and sorts them by key.
func entries(vars map[string]string) EnvList {
	list := EnvList{Entries: make([]EnvEntry, 0, len(vars))}
	for _, key := range slices.Sorted(maps.Keys(vars)) {
		list.Entries = append(list.Entries, EnvEntry{Key: key, Masked: mask(vars[key])})
	}
	return list
}

// checkPatch refuses a key that is not a variable name and a key both set and removed, before
// anything is written.
func checkPatch(patch EnvPatch) error {
	for key := range patch.Set {
		if !keyPattern.MatchString(key) {
			return troncerrors.Invalid(fmt.Sprintf("%q is not a variable name: use letters, digits and '_', "+
				"not starting with a digit", key))
		}
	}
	for _, key := range patch.Unset {
		if !keyPattern.MatchString(key) {
			return troncerrors.Invalid(fmt.Sprintf("%q is not a variable name", key))
		}
		if _, both := patch.Set[key]; both {
			return troncerrors.Invalid(fmt.Sprintf("%s is both set and removed", key))
		}
	}
	return nil
}
