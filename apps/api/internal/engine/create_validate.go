package engine

import (
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/repopath"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// validateCreate rejects a create request before anything is locked or cloned: a missing or
// invalid name, a missing repository, a container port out of range, a path that reaches outside
// the repository and an invalid tag. The messages name the CLI flags, which is where every request
// comes from today.
func validateCreate(req CreateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("app name is required as an argument or inferrable from --repo")
	}
	if err := appname.Validate(req.Name); err != nil {
		return err
	}
	if req.Repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if req.Flags.ContainerPort < 0 || req.Flags.ContainerPort > 65535 {
		return fmt.Errorf("container port must be between 1 and 65535, got %d", req.Flags.ContainerPort)
	}
	if err := validateCreatePaths(req); err != nil {
		return err
	}
	for _, tag := range req.Flags.Tags {
		if err := storage.ValidateTag(tag); err != nil {
			return err
		}
	}
	return nil
}

// validateCreatePaths rejects a --static, --context or --containerfile flag that would reach
// outside the repository, before anything is cloned.
func validateCreatePaths(req CreateRequest) error {
	return repopath.ValidateSet(map[string]string{
		"--static":        req.Flags.StaticDir,
		"--context":       req.Flags.Context,
		"--containerfile": req.Flags.Containerfile,
	})
}
