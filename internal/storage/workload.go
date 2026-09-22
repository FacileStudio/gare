package storage

import (
	"fmt"
	"strings"
)

// WorkloadType identifies how an application workload is supervised.
type WorkloadType string

const (
	WorkloadContainer WorkloadType = "container"
	WorkloadStatic    WorkloadType = "static"
	WorkloadCompose   WorkloadType = "compose"
)

// ResolveWorkload returns the configured workload type, rejecting unknown values.
func (g *GareFile) ResolveWorkload() (WorkloadType, error) {
	if g == nil {
		return WorkloadContainer, nil
	}
	switch strings.ToLower(strings.TrimSpace(g.Type)) {
	case "", string(WorkloadContainer):
		return WorkloadContainer, nil
	case string(WorkloadStatic):
		return WorkloadStatic, nil
	case string(WorkloadCompose):
		return WorkloadCompose, nil
	default:
		return "", fmt.Errorf("invalid type %q in gare configuration: want container, static, or compose", g.Type)
	}
}

// ResolveComposeFile returns the configured compose file name or empty when unset.
func (g *GareFile) ResolveComposeFile() string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(g.ComposeFile)
}

// IsStatic reports whether the application is configured as a static site.
func (c *AppConfig) IsStatic() bool {
	return c != nil && strings.EqualFold(c.AppType, string(WorkloadStatic))
}

// IsCompose reports whether the application is configured as a compose stack.
func (c *AppConfig) IsCompose() bool {
	return c != nil && strings.EqualFold(c.AppType, string(WorkloadCompose))
}

// UsesPodManifest reports whether the application is deployed with podman kube play.
func (c *AppConfig) UsesPodManifest() bool {
	return !c.IsStatic() && !c.IsCompose()
}

// UsesQuadletUnit reports whether the workload is supervised through a Quadlet source file.
func (c *AppConfig) UsesQuadletUnit() bool {
	return !c.IsCompose()
}

// UsesAppEnvFile reports whether environment variables live in the app env file.
func (c *AppConfig) UsesAppEnvFile() bool {
	return c.IsStatic() || c.IsCompose()
}
