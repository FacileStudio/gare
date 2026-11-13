// Package deploys serves an app's deploys over HTTP: starting one, rolling back, the history and
// one deploy's log as a server-sent event stream. A deploy goes through the webhook queue, so an API
// deploy and a pushed one of the same app coalesce and share the daemon's parallelism limit.
package deploys

import "github.com/FacileStudio/gare/apps/api/internal/deploys"

// Deploy is one deploy as record.json stores it, which is already the contract's shape.
type Deploy = deploys.Deploy

// DeployList is the answer of GET /api/apps/{name}/deploys, newest first.
type DeployList struct {
	Deploys []Deploy `json:"deploys"`
}

// RollbackRequest is the body of POST /api/apps/{name}/rollback. Without a deploy id the app goes
// back to its newest successful deploy of another commit.
type RollbackRequest struct {
	DeployID string `json:"deploy_id,omitempty"`
}
