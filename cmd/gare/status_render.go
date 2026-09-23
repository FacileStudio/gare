package main

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/FacileStudio/gare/internal/systemd"
)

func outputStatusHuman(w io.Writer, d *AppStatusDetails) error {
	bold := lipgloss.NewStyle().Bold(true)
	title := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPrimary)).Bold(true)

	root := tree.Root(bold.Render("Application: ") + title.Render(d.Name))
	root.Child(workloadSection(d, bold), sourceSection(d, bold))
	if d.Service != nil && d.Service.ActiveState != "" {
		root.Child(serviceSection(d.Service, bold))
	}

	fmt.Fprintln(w, root.String())
	return nil
}

// workloadSection renders the workload subtree. Every section is built as its own tree because
// tree.Child returns its receiver, so a section only nests when it is passed as a child itself.
func workloadSection(d *AppStatusDetails, bold lipgloss.Style) *tree.Tree {
	section := tree.Root(bold.Render("Workload"))
	section.Child("Type:   " + d.AppType)
	section.Child("Status: " + statusStyle(d.Status).Render(d.Status))
	if d.Port > 0 {
		section.Child(portLine(d))
	}
	if len(d.Domains) > 0 {
		section.Child("Domain: " + strings.Join(d.Domains, ", "))
	}
	if len(d.Tags) > 0 {
		section.Child("Tags:   " + strings.Join(d.Tags, ", "))
	}
	if d.Healthcheck != "" {
		section.Child("Health: " + d.Healthcheck)
	}
	appendProbes(section, d)
	return section
}

// appendProbes lists the individual probes only when there is more than one, because a single probe
// repeats the health path already shown above it.
func appendProbes(section *tree.Tree, d *AppStatusDetails) {
	if len(d.Probes) < 2 {
		return
	}
	for _, probe := range d.Probes {
		section.Child(fmt.Sprintf("Probe:  %s -> %s", probe.Label(), probeTarget(probe)))
	}
}

func portLine(d *AppStatusDetails) string {
	if d.ContainerPort > 0 && d.ContainerPort != d.Port {
		return fmt.Sprintf("Port:   %d -> %d", d.Port, d.ContainerPort)
	}
	return fmt.Sprintf("Port:   %d", d.Port)
}

func sourceSection(d *AppStatusDetails, bold lipgloss.Style) *tree.Tree {
	section := tree.Root(bold.Render("Source"))
	section.Child("Repo:   " + d.RepoURL)
	section.Child("Branch: " + d.Branch)
	section.Child("Commit: " + d.Commit)
	return section
}

func serviceSection(svc *systemd.ServiceProperties, bold lipgloss.Style) *tree.Tree {
	section := tree.Root(bold.Render("Systemd Service"))
	section.Child(fmt.Sprintf("State:   %s (%s)", svc.ActiveState, svc.SubState))
	if svc.MainPID > 0 {
		section.Child(fmt.Sprintf("Main PID: %d", svc.MainPID))
	}
	if svc.ActiveEnterTimestamp != "" {
		section.Child("Since:    " + svc.ActiveEnterTimestamp)
	}
	if svc.MemoryCurrent > 0 {
		section.Child("Memory:   " + formatMemory(svc.MemoryCurrent))
	}
	return section
}

func formatMemory(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
