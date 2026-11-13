package storage

import (
	"fmt"
	"strings"
)

// WorkloadType identifies how an application workload is supervised.
type WorkloadType string

const (
	WorkloadUnknown   WorkloadType = ""
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

// ResolveWorkloadType returns the workload type recorded on an application's configuration. It
// reports an error instead of defaulting to a container when the configuration names no usable
// type, so a caller that must not guess — destroy, which tears a running workload down — can tell an
// application that is a container workload from one whose type it does not know. The IsStatic and
// IsCompose predicates remain for callers that already hold a configuration.
func (c *AppConfig) ResolveWorkloadType() (WorkloadType, error) {
	if c == nil {
		return WorkloadUnknown, fmt.Errorf("no application configuration is loaded")
	}
	switch strings.ToLower(strings.TrimSpace(c.AppType)) {
	case "", string(WorkloadContainer):
		return WorkloadContainer, nil
	case string(WorkloadStatic):
		return WorkloadStatic, nil
	case string(WorkloadCompose):
		return WorkloadCompose, nil
	default:
		return WorkloadUnknown, fmt.Errorf("unknown workload type %q in the application configuration", c.AppType)
	}
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

// UsesAppEnvFile reports whether environment variables live in the app env file.
func (c *AppConfig) UsesAppEnvFile() bool {
	return c.IsStatic() || c.IsCompose()
}
