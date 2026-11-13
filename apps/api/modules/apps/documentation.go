package apps

import "github.com/FacileStudio/tronc/apiref"

// session is the auth every route here asks for.
const session = "session"

// Documentation is the module's entry in the API reference served at /docs.
func Documentation() apiref.Module {
	return apiref.Module{
		Name: "Apps",
		Description: "The applications gare manages. Every route needs a session; a cookie-authenticated " +
			"mutation must send X-Facile-CSRF. A change to an app another operation holds answers 409 after a " +
			"short wait instead of hanging.",
		Routes: append(append(readRoutes(), changeRoutes()...), lifecycleRoutes()...),
	}
}

// nameParam is the {name} path parameter.
func nameParam() []apiref.Field {
	return []apiref.Field{{Name: "name", Type: "string", Description: "the app's name"}}
}

// linesParam is the journal stream's ?lines.
func linesParam() []apiref.Field {
	return []apiref.Field{{Name: "lines", Type: "integer",
		Description: "how many past lines to start with, 1 to 2000, default 200"}}
}

// readRoutes are the routes that only read.
func readRoutes() []apiref.Route {
	notFound := []apiref.Error{{Status: 404, Code: "not_found", Description: "no app of that name"}}
	return []apiref.Route{
		{Method: "GET", Path: "/api/apps", Summary: "List the apps", Auth: session, ResponseBody: AppList{}},
		{Method: "GET", Path: "/api/apps/{name}", Summary: "Read one app", Auth: session, PathParams: nameParam(),
			Description:  "status is systemd's; rollback says whether an earlier image can be gone back to now.",
			ResponseBody: App{}, Errors: notFound},
		{Method: "GET", Path: "/api/apps/{name}/logs", Summary: "Follow the app's journal", Auth: session,
			PathParams: nameParam(), QueryParams: linesParam(),
			Description: "text/event-stream: one data field per journal line, until the client disconnects. " +
				"Answers 401 JSON without a session, never a redirect.",
			Errors: notFound},
	}
}

// changeRoutes create, destroy and retag an app.
func changeRoutes() []apiref.Route {
	return []apiref.Route{
		{Method: "POST", Path: "/api/apps", Summary: "Create an app", Auth: session, Status: 201,
			Description: "Clones the repository and writes the app's unit. repo_url must start with https://, " +
				"ssh:// or git@; neither it nor branch may start with '-'.",
			RequestBody: CreateRequest{}, ResponseBody: App{},
			Errors: []apiref.Error{{Status: 400, Code: "invalid_argument", Description: "a refused field or clone"},
				{Status: 409, Code: "already_exists", Description: "an app of that name exists"}}},
		destroyRoute(),
		{Method: "PUT", Path: "/api/apps/{name}/tags", Summary: "Replace the app's tags", Auth: session,
			PathParams: nameParam(), RequestBody: TagsRequest{}, ResponseBody: App{}},
	}
}

// destroyRoute is DELETE /api/apps/{name}.
func destroyRoute() apiref.Route {
	volumes := apiref.Field{Name: "volumes", Type: "boolean", Description: "true also removes a compose app's named volumes"}
	refused := apiref.Error{Status: 400, Code: "invalid_argument", Description: "confirm is not the name"}
	return apiref.Route{
		Method: "DELETE", Path: "/api/apps/{name}", Summary: "Destroy an app", Auth: session, Status: 204,
		PathParams: nameParam(), QueryParams: []apiref.Field{volumes},
		Description: "Stops and removes everything gare made for the app. It cannot be undone, so confirm " +
			"must repeat the name.",
		RequestBody: DestroyRequest{}, Errors: []apiref.Error{refused},
	}
}

// lifecycleRoutes start, stop and restart an app's unit.
func lifecycleRoutes() []apiref.Route {
	busy := []apiref.Error{{Status: 409, Code: "already_exists", Description: "another operation on the app is running"}}
	route := func(action, summary string) apiref.Route {
		return apiref.Route{Method: "POST", Path: "/api/apps/{name}/" + action, Summary: summary, Auth: session,
			PathParams: nameParam(), ResponseBody: App{}, Errors: busy}
	}
	return []apiref.Route{
		route(actionStart, "Start the app"),
		route(actionStop, "Stop the app"),
		route(actionRestart, "Restart the app"),
	}
}
