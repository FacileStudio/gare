package appname

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"a", false},
		{"my-app", false},
		{"my_app-123", false},
		{strings.Repeat("a", 63), false},
		{"", true},
		{"../etc", true},
		{"../../bad", true},
		{"-leading-dash", true},
		{"_leading-under", true},
		{"has spaces", true},
		{"invalid@char", true},
		{"invalid.dot", true},
		{strings.Repeat("a", 64), true},
	}
	for _, tc := range cases {
		err := Validate(tc.name)
		if (err != nil) != tc.wantErr {
			t.Errorf("Validate(%q) error = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}
