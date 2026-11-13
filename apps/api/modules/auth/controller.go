package auth

import "github.com/FacileStudio/porte/kit"

// controller shapes what the routes answer. It holds no state and touches no
// database, so the answer for one identity is always the same.
type controller struct{}

// me turns the request identity into the bare Me the client reads, with an
// absent picture as null so the client can draw initials instead.
func (controller) me(identity kit.Identity) Me {
	me := Me{Email: identity.Email, Name: identity.Name}
	if identity.PictureURL != "" {
		picture := identity.PictureURL
		me.Picture = &picture
	}
	return me
}
