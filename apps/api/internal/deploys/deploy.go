// Package deploys keeps the history of an app's deploys on disk: one directory per deploy holding
// its record and its log, so the CLI, the webhook daemon and the HTTP API agree on what ran, how it
// ended and what it printed, and a reader in another process sees the same history by reading files.
package deploys

import (
	"fmt"
	"regexp"
	"time"
)

// Status is where a deploy stands. A record moves from queued or running to one final status.
type Status string

const (
	// StatusQueued is a deploy accepted but waiting for its turn.
	StatusQueued Status = "queued"
	// StatusRunning is a deploy in progress.
	StatusRunning Status = "running"
	// StatusSucceeded is a deploy whose workload came up and passed its probes.
	StatusSucceeded Status = "succeeded"
	// StatusFailed is a deploy that stopped on an error and left nothing restored.
	StatusFailed Status = "failed"
	// StatusRolledBack is a deploy that failed its health probes and was replaced by an earlier one.
	StatusRolledBack Status = "rolled-back"
)

// Trigger names who started a deploy, so the history tells a typed deploy from a pushed one.
type Trigger string

const (
	// TriggerCLI is a deploy an operator typed.
	TriggerCLI Trigger = "cli"
	// TriggerWebhook is a deploy a git push started.
	TriggerWebhook Trigger = "webhook"
	// TriggerAPI is a deploy the HTTP API started.
	TriggerAPI Trigger = "api"
)

// Deploy is one deploy as it is stored in record.json and served by the API.
type Deploy struct {
	ID           string     `json:"id"`
	App          string     `json:"app"`
	Commit       string     `json:"commit"`
	Trigger      Trigger    `json:"trigger"`
	Status       Status     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Error        *string    `json:"error"`
	RolledBackTo *string    `json:"rolled_back_to"`
}

// Finished reports whether the deploy reached a final status, after which its log never grows.
func (d Deploy) Finished() bool {
	return d.Status != StatusQueued && d.Status != StatusRunning
}

// shortSHALength is how many hex digits of a commit name a deploy and its image tag: long enough
// that two commits of one repository never collide in practice.
const shortSHALength = 12

// ShortSHA returns the abbreviated commit a deploy id and an image tag carry, or "" for no commit.
func ShortSHA(commit string) string {
	if len(commit) > shortSHALength {
		return commit[:shortSHALength]
	}
	return commit
}

// idPattern is the only shape a deploy id takes. Ids arrive from HTTP paths, so anything else is
// refused before it is joined to a directory.
var idPattern = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{3}Z-[0-9a-z]+$`)

// ValidateID refuses a string that is not a deploy id, which keeps a crafted id from naming a path
// outside the app's history.
func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid deploy id %q", id)
	}
	return nil
}

// newID names a deploy from the time it was created and its short commit, "pending" while the
// commit is not known yet, so ids sort in creation order.
func newID(at time.Time, commit string) string {
	suffix := ShortSHA(commit)
	if suffix == "" {
		suffix = "pending"
	}
	return at.UTC().Format("20060102T150405.000Z") + "-" + suffix
}
