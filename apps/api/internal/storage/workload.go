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
// application that is a container workload from one whose type it does not know. workload.For
// selects an adapter from it, so no caller asks about the type again.
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
