package storage

import "testing"

func TestGareFileResolveNetwork(t *testing.T) {
	var nilGf *GareFile
	if got := nilGf.ResolvePort(); got != 0 {
		t.Errorf("nil ResolvePort: got %d, want 0", got)
	}
	if got := nilGf.ResolveDomain(); got != "" {
		t.Errorf("nil ResolveDomain: got %q, want empty", got)
	}

	gf := &GareFile{
		Port:   8080,
		Domain: "example.com",
	}
	if got := gf.ResolvePort(); got != 8080 {
		t.Errorf("ResolvePort: got %d, want 8080", got)
	}
	if got := gf.ResolveDomain(); got != "example.com" {
		t.Errorf("ResolveDomain: got %q, want example.com", got)
	}
}
