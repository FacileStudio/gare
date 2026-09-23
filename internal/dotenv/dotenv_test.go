package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBasic(t *testing.T) {
	content := "FOO=bar\nBAZ=qux\n"
	envs, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envs["FOO"] != "bar" || envs["BAZ"] != "qux" {
		t.Errorf("unexpected envs: %+v", envs)
	}
}

func TestParseQuotesAndComments(t *testing.T) {
	content := "A=\"hello world\" # comment\nB='single quoted' # comment\nC=simple # inline\nexport D=exported\n"
	envs, err := Parse(content)
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

func TestParseMultiline(t *testing.T) {
	content := "MULTI=\"line1\nline2\"\nOTHER=val\n"
	envs, err := Parse(content)
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

func TestLoadFile(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	content := "# Comment\n\nFOO=bar\nexport BAZ=\"spaces\"\nSINGLE='single'\nINVALID\n"
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	vars, err := LoadFile(envPath)
	if err != nil || len(vars) != 3 {
		t.Fatalf("expected 3 vars, got %d, err: %v", len(vars), err)
	}
	if vars["FOO"] != "bar" || vars["BAZ"] != "spaces" || vars["SINGLE"] != "single" {
		t.Errorf("unexpected vars values: %+v", vars)
	}
}

func TestParseAssignments(t *testing.T) {
	args := []string{"KEY1=VAL1", "KEY2=VAL2=EXTRA"}
	vars, err := ParseAssignments(args)
	if err != nil || len(vars) != 2 || vars["KEY2"] != "VAL2=EXTRA" {
		t.Fatalf("ParseAssignments mismatch, got %+v, err: %v", vars, err)
	}

	if _, err := ParseAssignments([]string{"INVALID"}); err == nil {
		t.Errorf("expected error for invalid assignment, got nil")
	}
}
