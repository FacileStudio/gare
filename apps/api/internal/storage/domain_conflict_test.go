package storage

import "testing"

func TestDomainsConflict(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"example.com", "Example.com:443", true},
		{" EXAMPLE.com ", "example.com", true},
		{"example.com", "example.com:80", true},
		{"example.com", "example.com:8443", true},
		{"example.com:8443", "example.com:8443", true},
		{"example.com:443", "example.com:80", true},
		{"example.com:8443", "example.com:9443", false},
		{"example.com", "www.example.com", false},
	}
	for _, c := range cases {
		if got := DomainsConflict(c.a, c.b); got != c.want {
			t.Errorf("DomainsConflict(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestAddDomainRejectsTheSameSiteWithItsDefaultPort(t *testing.T) {
	cfg := &AppConfig{Name: "app", Domains: []string{"example.com"}}
	if err := cfg.AddDomain("example.com:443"); err == nil {
		t.Fatal("expected example.com:443 to conflict with example.com on the same app")
	}
}
