package apps

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// Journal stream bounds: how many past lines a stream starts with when the client names no count,
// and the most it may ask for, so one request cannot make journalctl replay a whole journal.
const (
	defaultLogLines = 200
	maxLogLines     = 2000
)

// controller validates requests and shapes answers. It holds no state but whether a create may
// clone from a local path, and runs no command.
type controller struct {
	allowLocal bool
}

// createRequest turns a create body into the engine's request, refusing what must never reach git:
// a repository or branch that starts with "-", which git would read as an option, a URL of another
// scheme, an unknown type and a port out of range.
func (c controller) createRequest(req CreateRequest) (engine.CreateRequest, error) {
	if err := c.checkRepo(req.RepoURL); err != nil {
		return engine.CreateRequest{}, err
	}
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}
	if strings.HasPrefix(branch, "-") || strings.ContainsAny(branch, " \t\r\n") {
		return engine.CreateRequest{}, troncerrors.Invalid("branch must not start with '-' or contain spaces")
	}
	if !slices.Contains([]string{"", "container", "static", "compose"}, req.Type) {
		return engine.CreateRequest{}, troncerrors.Invalid("type must be container, static or compose")
	}
	if req.Port < 0 || req.Port > 65535 {
		return engine.CreateRequest{}, troncerrors.Invalid("port must be between 1 and 65535")
	}
	name := req.Name
	if name == "" {
		name = engine.DeriveName(req.RepoURL)
	}
	if err := appname.Validate(name); err != nil {
		return engine.CreateRequest{}, troncerrors.Invalid(fmt.Sprintf("%q cannot name an app: use letters, digits, "+
			"'-' and '_', starting with a letter or digit", name))
	}
	flags := appconfig.Flags{Type: req.Type, Port: req.Port}
	return engine.CreateRequest{Name: name, Repo: req.RepoURL, Branch: branch, Flags: flags}, nil
}

// checkRepo accepts an https, ssh or scp-style git URL, and a local absolute path only when tests
// allow it.
func (c controller) checkRepo(repo string) error {
	switch {
	case repo == "":
		return troncerrors.Invalid("repo_url is required")
	case strings.HasPrefix(repo, "-"):
		return troncerrors.Invalid("repo_url must not start with '-'")
	case strings.ContainsAny(repo, " \t\r\n"):
		return troncerrors.Invalid("repo_url must not contain spaces")
	}
	for _, prefix := range []string{"https://", "ssh://", "git@"} {
		if strings.HasPrefix(repo, prefix) {
			return nil
		}
	}
	if c.allowLocal && filepath.IsAbs(repo) {
		return nil
	}
	return troncerrors.Invalid("repo_url must start with https://, ssh:// or git@")
}

// app shapes an app's status, newest deploy and rollback state into the answer the client reads.
func (controller) app(st *engine.AppStatus, last *deploys.Deploy, rollback bool) App {
	commit := st.Commit
	if commit == "-" {
		commit = ""
	}
	return App{
		Name: st.Name, Type: st.AppType, Status: statusOf(st.Status), RepoURL: st.RepoURL, Branch: st.Branch,
		Commit: commit, Port: st.Port, Domains: orEmpty(st.Domains), Tags: orEmpty(st.Tags),
		CreatedAt: st.CreatedAt, LastDeploy: last, Rollback: rollback,
	}
}

// statusOf folds systemd's active state into the states the client draws; anything else, such as
// reloading or maintenance, is unknown.
func statusOf(state string) string {
	if slices.Contains([]string{"active", "activating", "deactivating", "inactive", "failed"}, state) {
		return state
	}
	return "unknown"
}

// orEmpty answers an absent list as [] rather than null, so the client can iterate it.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// logLines reads the lines query parameter of the journal stream.
func logLines(raw string) (int, error) {
	if raw == "" {
		return defaultLogLines, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > maxLogLines {
		return 0, troncerrors.Invalid(fmt.Sprintf("lines must be a number from 1 to %d", maxLogLines))
	}
	return n, nil
}

// tagChanges returns the tags to add and to remove for an app holding current to end with wanted,
// after refusing an invalid one.
func tagChanges(current, wanted []string) ([]string, []string, error) {
	keep := map[string]bool{}
	var add, remove []string
	for _, tag := range wanted {
		if err := storage.ValidateTag(tag); err != nil {
			return nil, nil, troncerrors.Invalid(err.Error())
		}
		keep[storage.NormalizeTag(tag)] = true
		add = append(add, storage.NormalizeTag(tag))
	}
	for _, tag := range current {
		if !keep[storage.NormalizeTag(tag)] {
			remove = append(remove, tag)
		}
	}
	return add, remove, nil
}
