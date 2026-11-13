// Package apps serves the managed applications over HTTP: the list and one app's state, create and
// destroy, start, stop and restart, the tags, and the runtime journal as a stream. Every handler
// drives the engine; none runs a command of its own.
package apps

import "github.com/FacileStudio/gare/apps/api/internal/deploys"

// App is one application as the client reads it. Status is systemd's, folded into the states the
// client knows; Rollback says whether an earlier image can be gone back to now.
type App struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Status     string          `json:"status"`
	RepoURL    string          `json:"repo_url"`
	Branch     string          `json:"branch"`
	Commit     string          `json:"commit"`
	Port       int             `json:"port"`
	Domains    []string        `json:"domains"`
	Tags       []string        `json:"tags"`
	CreatedAt  string          `json:"created_at"`
	LastDeploy *deploys.Deploy `json:"last_deploy"`
	Rollback   bool            `json:"rollback"`
}

// AppList is the answer of GET /api/apps.
type AppList struct {
	Apps []App `json:"apps"`
}

// CreateRequest is the body of POST /api/apps. Name is derived from the repository when empty,
// Branch defaults to main and Type to what the repository's gare.yml says.
type CreateRequest struct {
	Name    string `json:"name,omitempty"`
	RepoURL string `json:"repo_url"`
	Branch  string `json:"branch,omitempty"`
	Type    string `json:"type,omitempty"`
	Port    int    `json:"port,omitempty"`
}

// DestroyRequest is the body of DELETE /api/apps/{name}: the app's name typed back, because a
// destroy cannot be undone.
type DestroyRequest struct {
	Confirm string `json:"confirm"`
}

// TagsRequest is the body of PUT /api/apps/{name}/tags: the whole set of tags the app ends with.
type TagsRequest struct {
	Tags []string `json:"tags"`
}
