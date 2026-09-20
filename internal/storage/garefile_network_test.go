package storage

import "testing"

func TestGareFileResolveNetwork(t *testing.T) {
	var nilGf *GareFile
	if got := nilGf.ResolvePort(); got != 0 {
		t.Errorf("nil ResolvePort: got %d, want 0", got)
	}

	gf := &GareFile{
		Port: 8080,
		Tags: []string{"client-a", "prod"},
	}
	if got := gf.ResolvePort(); got != 8080 {
		t.Errorf("ResolvePort: got %d, want 8080", got)
	}
	if got := gf.ResolveTags(); len(got) != 2 || got[0] != "client-a" {
		t.Errorf("ResolveTags: unexpected %v", got)
	}
	if got := nilGf.ResolveTags(); got != nil {
		t.Errorf("nil ResolveTags: got %v, want nil", got)
	}
}
