package auth

import (
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/apiref"
)

// Documentation is the module's entry in the API reference served at /docs.
func Documentation() apiref.Module {
	return apiref.Module{
		Name: "Auth",
		Description: "Sign-in through porte against the suite's identity provider, single sign-on only. " +
			"A session is an HttpOnly cookie; a cookie-authenticated mutation must send X-Facile-CSRF. " +
			"Only the addresses in GARE_ADMIN_EMAILS can sign in, and a session stops working once its " +
			"address leaves the list.",
		Routes: append(append(signInRoutes(), sessionRoutes()...), meRoute()),
	}
}

// signInRoutes are the browser navigations of the OIDC flow.
func signInRoutes() []apiref.Route {
	refused := "Every refusal, including an address outside GARE_ADMIN_EMAILS, is a 302 to /login?error=<message>."
	return []apiref.Route{
		{Method: "GET", Path: "/api/auth/config", Summary: "Read what this instance accepts", Auth: "none",
			ResponseBody: porte.ConfigResponse{}},
		{Method: "GET", Path: "/api/auth/oidc", Summary: "Start a sign-in", Auth: "none", Status: 302,
			Description: "Redirects to the identity provider with PKCE, a nonce and a state cookie."},
		{Method: "GET", Path: "/api/auth/oidc/callback", Summary: "Finish a sign-in", Auth: "none", Status: 302,
			Description: "Sets the session cookie and redirects to the app. " + refused},
		{Method: "POST", Path: "/api/auth/oidc/exchange", Summary: "Trade a CLI login code", Auth: "none",
			Description:  "Mounted by porte. gare has no CLI sign-in and mints no code, so every call is a 401.",
			RequestBody:  porte.ExchangeRequest{},
			ResponseBody: porte.ExchangeResponse{},
			Errors:       []apiref.Error{{Status: 401, Code: "unauthenticated", Description: "no such code"}}},
	}
}

// sessionRoutes end or refresh a session, or let the provider end them.
func sessionRoutes() []apiref.Route {
	return []apiref.Route{
		{Method: "POST", Path: "/api/auth/logout", Summary: "Sign out", Auth: "session or none",
			Description:  "Idempotent: answers 200 with or without a live session. A cookie caller must send X-Facile-CSRF.",
			ResponseBody: porte.LogoutResponse{},
			Errors:       []apiref.Error{{Status: 403, Code: "permission_denied", Description: "cookie without X-Facile-CSRF"}}},
		{Method: "POST", Path: "/api/auth/sync-profile", Summary: "Refresh name and picture", Auth: "session",
			ResponseBody: porte.SyncProfileResponse{}},
		{Method: "POST", Path: "/api/auth/backchannel-logout", Summary: "Provider-initiated sign-out", Auth: "logout token",
			Description:  "OpenID Connect back-channel logout: revokes every session of the subject.",
			ResponseBody: porte.LogoutResponse{}},
		{Method: "POST", Path: "/api/registre/deletions", Summary: "Provider erasure notice", Auth: "signed notice",
			Description: "The provider's signed user.deleted push. The account and its sessions are erased."},
	}
}

// meRoute is the signed-in operator.
func meRoute() apiref.Route {
	return apiref.Route{
		Method: "GET", Path: "/api/me", Summary: "Read the signed-in operator", Auth: "session",
		ResponseBody: Me{},
		Errors:       []apiref.Error{{Status: 401, Code: "unauthenticated", Description: "no session, or the address left GARE_ADMIN_EMAILS"}},
	}
}
