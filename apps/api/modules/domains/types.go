// Package domains serves the hostnames an app is routed on over HTTP. A change goes through the
// engine, which refuses a hostname another app owns and updates a running app's ingress at once.
package domains

// DomainList is the answer of every domain route: the app's hostnames afterwards.
type DomainList struct {
	Domains []string `json:"domains"`
}

// AddRequest is the body of POST /api/apps/{name}/domains.
type AddRequest struct {
	Hostname string `json:"hostname"`
}
