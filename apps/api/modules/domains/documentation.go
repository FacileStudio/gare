package domains

import "github.com/FacileStudio/tronc/apiref"

// Documentation is the module's entry in the API reference served at /docs.
func Documentation() apiref.Module {
	name := apiref.Field{Name: "name", Type: "string", Description: "the app's name"}
	hostname := apiref.Field{Name: "hostname", Type: "string", Description: "the hostname to stop routing"}
	taken := apiref.Error{Status: 409, Code: "already_exists", Description: "an app already owns the hostname"}
	missing := apiref.Error{Status: 404, Code: "not_found", Description: "no such app, or the app is not on it"}
	return apiref.Module{
		Name: "Domains",
		Description: "The hostnames Caddy routes to an app. A running app's ingress changes at once; a stopped " +
			"app picks the change up when it starts.",
		Routes: []apiref.Route{
			{Method: "GET", Path: "/api/apps/{name}/domains", Summary: "List the app's hostnames", Auth: "session",
				PathParams: []apiref.Field{name}, ResponseBody: DomainList{}},
			{Method: "POST", Path: "/api/apps/{name}/domains", Summary: "Add a hostname", Auth: "session",
				PathParams: []apiref.Field{name}, RequestBody: AddRequest{}, ResponseBody: DomainList{},
				Errors: []apiref.Error{taken}},
			{Method: "DELETE", Path: "/api/apps/{name}/domains/{hostname}", Summary: "Remove a hostname",
				Auth: "session", PathParams: []apiref.Field{name, hostname}, ResponseBody: DomainList{},
				Errors: []apiref.Error{missing}},
		},
	}
}
