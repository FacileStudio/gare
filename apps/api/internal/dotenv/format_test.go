package dotenv

import (
	"strings"
	"testing"
)

func roundTripValues() map[string]string {
	return map[string]string{
		"BARE":      "postgres://u:p@db:5432/x?sslmode=disable",
		"HASH":      "ab #cd",
		"QUOTED":    `"quoted"`,
		"LEAD":      " lead",
		"MULTI":     "line1\nline2",
		"BACKSLASH": `a\nb\\c`,
		"DOLLAR":    "$HOME and `cmd`",
		"SINGLE":    "it's",
		"EMPTY":     "",
		"UNICODE":   "café ☕",
	}
}

func TestFormatRoundTrips(t *testing.T) {
	want := roundTripValues()
	got, err := Parse(string(Format(want)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: wrote %q, read back %q", k, v, got[k])
		}
	}
	if len(got) != len(want) {
		t.Errorf("read %d keys, wrote %d", len(got), len(want))
	}
}

func TestFormatWritesSafeValuesBare(t *testing.T) {
	out := string(Format(map[string]string{"B": "x", "A": "https://a.b/c"}))
	if out != "A=https://a.b/c\nB=x\n" {
		t.Errorf("unexpected output %q", out)
	}
}

func TestParseFollowsSystemdEscapes(t *testing.T) {
	envs, err := Parse("A=\"x\\\"y\\\\\"\nB=\"keep\\n\"\nexporter=1\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if envs["A"] != `x"y\` {
		t.Errorf("A: got %q", envs["A"])
	}
	if envs["B"] != `keep\n` {
		t.Errorf("B: got %q", envs["B"])
	}
	if envs["exporter"] != "1" {
		t.Errorf("a key starting with export was mangled: %v", strings.Join(keysOf(envs), ","))
	}
}

func keysOf(envs map[string]string) []string {
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	return keys
}
