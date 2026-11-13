package engine

import (
	"strings"
	"testing"
)

func TestValidateCreateInputs(t *testing.T) {
	testValidInputs(t)
	testInvalidNames(t)
	testMissingRequired(t)
}

func testValidInputs(t *testing.T) {
	valid := []string{
		"myapp",
		"my-app-1",
		"app_test",
		"a",
		"web-service",
	}
	for _, name := range valid {
		if err := validateCreate(CreateRequest{Name: name, Repo: "https://git.example.com/repo"}); err != nil {
			t.Errorf("expected %s to be valid, got: %v", name, err)
		}
	}
}

func testInvalidNames(t *testing.T) {
	invalidNames := []string{
		"../traversal",
		"bad/slash",
		"has space",
		"-startshyphen",
		"_startsunderscore",
		"",
		strings.Repeat("a", 64),
		"bad$char",
	}
	for _, name := range invalidNames {
		if err := validateCreate(CreateRequest{Name: name, Repo: "https://git.example.com/repo"}); err == nil {
			t.Errorf("expected invalid name %q to fail validation", name)
		}
	}
}

func testMissingRequired(t *testing.T) {
	if err := validateCreate(CreateRequest{Name: "myapp"}); err == nil {
		t.Error("expected error when --repo is empty")
	}
}
