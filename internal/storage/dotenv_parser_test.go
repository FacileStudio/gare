package storage

import (
	"testing"
)

func TestParseDotEnvBasic(t *testing.T) {
	content := "FOO=bar\nBAZ=qux\n"
	envs, err := ParseDotEnv(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envs["FOO"] != "bar" || envs["BAZ"] != "qux" {
		t.Errorf("unexpected envs: %+v", envs)
	}
}

func TestParseDotEnvQuotesAndComments(t *testing.T) {
	content := "A=\"hello world\" # comment\nB='single quoted' # comment\nC=simple # inline\nexport D=exported\n"
	envs, err := ParseDotEnv(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envs["A"] != "hello world" {
		t.Errorf("expected hello world, got %q", envs["A"])
	}
	if envs["B"] != "single quoted" {
		t.Errorf("expected single quoted, got %q", envs["B"])
	}
	if envs["C"] != "simple" {
		t.Errorf("expected simple, got %q", envs["C"])
	}
	if envs["D"] != "exported" {
		t.Errorf("expected exported, got %q", envs["D"])
	}
}

func TestParseDotEnvMultiline(t *testing.T) {
	content := "MULTI=\"line1\nline2\"\nOTHER=val\n"
	envs, err := ParseDotEnv(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envs["MULTI"] != "line1\nline2" {
		t.Errorf("expected multiline value, got %q", envs["MULTI"])
	}
	if envs["OTHER"] != "val" {
		t.Errorf("expected val, got %q", envs["OTHER"])
	}
}
