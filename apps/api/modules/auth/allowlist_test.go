package auth

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/FacileStudio/porte"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// countingUsers is a porte.UserStore that records how often porte's upsert ran.
type countingUsers struct{ calls int }

// UpsertFromOIDC accepts everyone as user 7 and counts the call.
func (c *countingUsers) UpsertFromOIDC(context.Context, porte.Claims) (int64, error) {
	c.calls++
	return 7, nil
}

// TestParseEmails proves the list is trimmed, lowercased and free of blanks.
func TestParseEmails(t *testing.T) {
	got := ParseEmails(" Ops@Example.com, ,dev@example.com ,")
	want := []string{"ops@example.com", "dev@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseEmails = %q, want %q", got, want)
	}
	if got := ParseEmails(""); len(got) != 0 {
		t.Fatalf("ParseEmails(\"\") = %q, want nothing", got)
	}
}

// TestAllowlistMatchesExactAddressesOnly proves case and spacing do not
// matter, a prefix or an empty address never matches, and an empty list
// lets nobody in.
func TestAllowlistMatchesExactAddressesOnly(t *testing.T) {
	list := newAllowlist([]string{"Ops@Example.com"})
	for email, want := range map[string]bool{
		"ops@example.com": true, " OPS@example.COM ": true,
		"ops@example.co": false, "": false, "dev@example.com": false,
	} {
		if got := list.allows(email); got != want {
			t.Errorf("allows(%q) = %v, want %v", email, got, want)
		}
	}
	if newAllowlist(nil).allows("ops@example.com") {
		t.Fatal("an empty list let someone in")
	}
}

// TestGateRefusesAfterTheSameWorkAsAnAcceptance proves porte's upsert runs
// for every sign-in, and that only a verified address on the list passes.
func TestGateRefusesAfterTheSameWorkAsAnAcceptance(t *testing.T) {
	cases := []struct {
		claims porte.Claims
		allow  bool
	}{
		{porte.Claims{Email: "ops@example.com", EmailVerified: true}, true},
		{porte.Claims{Email: "ops@example.com", EmailVerified: false}, false},
		{porte.Claims{Email: "stranger@example.com", EmailVerified: true}, false},
	}
	for _, c := range cases {
		inner := &countingUsers{}
		gate := gatedUsers{inner: inner, allow: newAllowlist([]string{"ops@example.com"})}
		userID, err := gate.UpsertFromOIDC(context.Background(), c.claims)
		if inner.calls != 1 {
			t.Fatalf("%+v: porte's upsert ran %d times, want 1", c.claims, inner.calls)
		}
		var apiErr *troncerrors.Error
		refused := errors.As(err, &apiErr) && apiErr.Code == "permission_denied"
		if c.allow != (err == nil && userID == 7) || c.allow == refused {
			t.Fatalf("%+v: got user %d, err %v; want allowed=%v", c.claims, userID, err, c.allow)
		}
	}
}
