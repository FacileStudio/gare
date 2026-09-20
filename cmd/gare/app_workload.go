package main

import (
	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/storage"
)

// writeWorkloadArtifacts writes the workload artifacts for the resolved type.
func writeWorkloadArtifacts(name, appDir string, opts appCreateOptions) error {
	switch opts.appType {
	case string(storage.WorkloadStatic):
		return writeStaticArtifacts(name, appDir, opts)
	case string(storage.WorkloadCompose):
		return writeComposeArtifacts(name, appDir, opts)
	default:
		return writeAppArtifacts(name, appDir, opts)
	}
}

// normalizeAppType infers the workload type when it is not set explicitly.
func normalizeAppType(opts appCreateOptions) string {
	if opts.appType != "" {
		return opts.appType
	}
	if opts.staticDir != "" {
		return "static"
	}
	return "container"
}

// normalizeCreateOptions applies workload-specific defaults and validates them.
func normalizeCreateOptions(repoDir string, opts appCreateOptions) (appCreateOptions, error) {
	opts.appType = normalizeAppType(opts)
	if opts.appType == "compose" {
		composeFile, err := resolveComposeWorkload(repoDir, opts.composeFile, opts.port)
		if err != nil {
			return opts, err
		}
		opts.composeFile = composeFile
	}
	if err := storage.ValidateHealthProbes(opts.healthProbes); err != nil {
		return opts, err
	}
	warnStrayComposeFile(repoDir, opts.appType == "compose")
	if opts.appType == "static" && opts.staticDir == "" {
		opts.staticDir = "."
	}
	if opts.appType == "container" {
		opts.containerPort = detectContainerPort(repoDir, opts)
	}
	return opts, nil
}

// detectContainerPort returns the configured container port or detects it from the containerfile.
func detectContainerPort(repoDir string, opts appCreateOptions) int {
	if opts.containerPort != 0 {
		return opts.containerPort
	}
	return builder.DetectExposedPort(repoDir, opts.containerfile)
}
