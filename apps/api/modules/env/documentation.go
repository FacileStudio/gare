package env

import "github.com/FacileStudio/tronc/apiref"

// Documentation is the module's entry in the API reference served at /docs.
func Documentation() apiref.Module {
	name := []apiref.Field{{Name: "name", Type: "string", Description: "the app's name"}}
	reveal := []apiref.Field{{Name: "reveal", Type: "string", Description: "a key: answer that variable in clear"}}
	missing := apiref.Error{Status: 404, Code: "not_found", Description: "no such app, or no such variable to reveal"}
	return apiref.Module{
		Name: "Env",
		Description: "An app's environment variables, in the store its workload reads. Values are masked: the first " +
			"and last two characters of a value of 12 characters or more, bullets otherwise.",
		Routes: []apiref.Route{
			{Method: "GET", Path: "/api/apps/{name}/env", Summary: "List or reveal variables", Auth: "session",
				PathParams: name, QueryParams: reveal, ResponseBody: EnvList{}, Errors: []apiref.Error{missing},
				Description: "Answers {entries} sorted by key, or {key, value} with ?reveal=KEY."},
			{Method: "PATCH", Path: "/api/apps/{name}/env", Summary: "Change variables", Auth: "session",
				PathParams: name, RequestBody: EnvPatch{}, ResponseBody: EnvList{},
				Description: "Merges: set adds or overwrites, unset removes, every other variable stays. A running app " +
					"restarts to read them."},
		},
	}
}
