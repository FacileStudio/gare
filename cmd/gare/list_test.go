package main

import (
	"testing"
	"time"
)

func TestRenderCreatedAtUsesTheLocalTimezone(t *testing.T) {
	if _, offset := time.Now().Zone(); offset == 0 {
		t.Skip("host runs in UTC, where the stored timestamp and its local rendering coincide")
	}
	if got := renderCreatedAt("2026-09-23T02:44:52Z"); got == "2026-09-23 02:44" {
		t.Errorf("expected the stored UTC timestamp in local time, got %s", got)
	}
}

func TestRenderCreatedAtFallsBackToTheStoredValue(t *testing.T) {
	if got := renderCreatedAt("not-a-timestamp"); got != "not-a-timestamp" {
		t.Errorf("expected an unparseable timestamp to be shown as stored, got %q", got)
	}
}
