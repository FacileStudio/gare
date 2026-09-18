package storage

import "testing"

func TestGareFileResolveNetwork(t *testing.T) {
	var nilGf *GareFile
	if got := nilGf.ResolvePort(); got != 0 {
		t.Errorf("nil ResolvePort: got %d, want 0", got)
	}

	gf := &GareFile{
		Port: 8080,
	}
	if got := gf.ResolvePort(); got != 8080 {
		t.Errorf("ResolvePort: got %d, want 8080", got)
	}
}
