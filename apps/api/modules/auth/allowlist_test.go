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

// FindByEmail knows only ops@example.com, as an account that already existed.
func (c *countingUsers) FindByEmail(_ context.Context, email string) (int64, error) {
	if email == "ops@example.com" {
		return 7, nil
	}
	return 0, porte.ErrNotFound
}

// knownIdentities is an identityFinder holding one existing subject.
type knownIdentities struct{}

// Find knows only the subject "known".
func (knownIdentities) Find(_ context.Context, _, subject string) (porte.StoredIdentity, error) {
	if subject == "known" {
		return porte.StoredIdentity{UserID: 7, Subject: subject}, nil
	}
	return porte.StoredIdentity{}, porte.ErrNotFound
}

// TestARefusalForgetsOnlyWhatItCreated proves the gate hands its cleanup
// exactly the rows the refused login created, and nothing for an acceptance:
// an existing subject or account is never marked new.
func TestARefusalForgetsOnlyWhatItCreated(t *testing.T) {
	cases := []struct {
		subject, email string
		want           *loginRows
	}{
		{"new", "stranger@example.com", &loginRows{userID: 7, subject: "new", newLink: true, newAccount: true}},
		{"known", "stranger@example.com", &loginRows{userID: 7, subject: "known", newAccount: true}},
		{"new", "ops@example.com", nil},
	}
	for _, c := range cases {
		var forgotten *loginRows
		gate := gatedUsers{inner: &countingUsers{}, identities: knownIdentities{},
			allow: newAllowlist([]string{"ops@example.com"}),
			forget: func(_ context.Context, rows loginRows) error {
				forgotten = &rows
				return nil
			}}
		_, err := gate.UpsertFromOIDC(context.Background(), porte.Claims{Subject: c.subject, Email: c.email, EmailVerified: true})
		if (err == nil) != (c.want == nil) || !reflect.DeepEqual(forgotten, c.want) {
			t.Fatalf("%s/%s: forgot %+v, want %+v", c.subject, c.email, forgotten, c.want)
		}
	}
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
		gate := gatedUsers{inner: inner, identities: knownIdentities{}, allow: newAllowlist([]string{"ops@example.com"}),
			forget: func(context.Context, loginRows) error { return nil }}
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
