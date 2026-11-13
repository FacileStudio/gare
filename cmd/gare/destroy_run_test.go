package main

import (
	"io"
	"strings"
	"testing"
)

func TestConfirmDestroy(t *testing.T) {
	cases := []struct {
		name        string
		yes         bool
		interactive bool
		input       string
		wantErr     bool
	}{
		{"--yes skips the prompt", true, false, "", false},
		{"no terminal and no --yes refuses", false, false, "myapp\n", true},
		{"typed name confirms", false, true, "myapp\n", false},
		{"typed name with spaces confirms", false, true, "  myapp \n", false},
		{"wrong name refuses", false, true, "other\n", true},
		{"empty answer refuses", false, true, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := confirmDestroy("myapp", tc.yes, strings.NewReader(tc.input), io.Discard, tc.interactive)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
