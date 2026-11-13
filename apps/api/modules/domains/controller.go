package domains

import (
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// domainList answers an app's hostnames, [] rather than null when it has none.
func domainList(cfg *storage.AppConfig) DomainList {
	if cfg.Domains == nil {
		return DomainList{Domains: []string{}}
	}
	return DomainList{Domains: cfg.Domains}
}

// checkHostname refuses a hostname Caddy could not serve, or one that could break out of a
// snippet, before the app's lock is taken.
func checkHostname(hostname string) error {
	if err := storage.ValidateDomain(hostname); err != nil {
		return troncerrors.Invalid(err.Error())
	}
	return nil
}
