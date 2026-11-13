// Package env serves an app's environment variables over HTTP. Values are masked unless one is
// asked for by name, and a change merges into the variables the app has: it never replaces the
// whole set, so a client that only knows some keys cannot erase the others.
package env

// EnvEntry is one variable with its value masked.
type EnvEntry struct {
	Key    string `json:"key"`
	Masked string `json:"masked"`
}

// EnvList is the answer of GET and PATCH /api/apps/{name}/env, sorted by key.
type EnvList struct {
	Entries []EnvEntry `json:"entries"`
}

// Revealed is the answer of GET /api/apps/{name}/env?reveal=KEY: one variable in clear.
type Revealed struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// EnvPatch is the body of PATCH /api/apps/{name}/env: variables to set or overwrite, and keys to
// remove. A key may not be in both.
type EnvPatch struct {
	Set   map[string]string `json:"set,omitempty"`
	Unset []string          `json:"unset,omitempty"`
}
