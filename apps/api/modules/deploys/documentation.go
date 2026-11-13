package deploys

import "github.com/FacileStudio/tronc/apiref"

// session is the auth every route here asks for.
const session = "session"

// Documentation is the module's entry in the API reference served at /docs.
func Documentation() apiref.Module {
	return apiref.Module{
		Name: "Deploys",
		Description: "Deploys and rollbacks of one app, and their logs. A deploy joins the same queue as webhook " +
			"pushes: one runs and one waits per app, and a request while one waits answers that waiting deploy.",
		Routes: append(changeRoutes(), readRoutes()...),
	}
}

// pathParams are the {name} and, when withID, {id} path parameters.
func pathParams(withID bool) []apiref.Field {
	fields := []apiref.Field{{Name: "name", Type: "string", Description: "the app's name"}}
	if withID {
		fields = append(fields, apiref.Field{Name: "id", Type: "string", Description: "the deploy's id"})
	}
	return fields
}

// changeRoutes start a deploy or a rollback.
func changeRoutes() []apiref.Route {
	refused := apiref.Error{Status: 412, Code: "failed_precondition",
		Description: "the app cannot be rolled back: not a container app, or no earlier image is kept"}
	busy := apiref.Error{Status: 409, Code: "already_exists", Description: "another operation on the app is running"}
	return []apiref.Route{
		{Method: "POST", Path: "/api/apps/{name}/deploy", Summary: "Deploy the app", Auth: session, Status: 202,
			PathParams: pathParams(false), ResponseBody: Deploy{},
			Description: "Queues a deploy and answers its record, queued or already running. It runs in the background."},
		{Method: "POST", Path: "/api/apps/{name}/rollback", Summary: "Roll the app back", Auth: session, Status: 202,
			PathParams: pathParams(false), RequestBody: RollbackRequest{}, ResponseBody: Deploy{},
			Description: "Runs the image an earlier successful deploy built, container apps only. The answer is the " +
				"rollback's own record, whose rolled_back_to names the deploy it restores.",
			Errors: []apiref.Error{refused, busy}},
	}
}

// readRoutes read the history and follow a log.
func readRoutes() []apiref.Route {
	missing := []apiref.Error{{Status: 404, Code: "not_found", Description: "no such app or deploy"}}
	return []apiref.Route{
		{Method: "GET", Path: "/api/apps/{name}/deploys", Summary: "List the app's deploys", Auth: session,
			PathParams: pathParams(false), ResponseBody: DeployList{}, Description: "Newest first, the last 50."},
		{Method: "GET", Path: "/api/apps/{name}/deploys/{id}", Summary: "Read one deploy", Auth: session,
			PathParams: pathParams(true), ResponseBody: Deploy{}, Errors: missing},
		{Method: "GET", Path: "/api/apps/{name}/deploys/{id}/log", Summary: "Follow one deploy's log", Auth: session,
			PathParams: pathParams(true), Errors: missing,
			Description: "Always text/event-stream: the whole log from its first line, one data field per line with " +
				"id set to the line number, then new lines while the deploy runs, then event done with the final " +
				"Deploy as JSON. Last-Event-ID resumes after that line. Answers 401 JSON without a session."},
	}
}
