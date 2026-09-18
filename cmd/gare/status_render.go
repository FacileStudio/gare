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

	header := bold.Render("Application: ") + title.Render(d.Name)
	t := tree.New().Root(header)

	appendWorkloadDetails(t, d, bold)
	appendSourceDetails(t, d, bold)

	if d.Service != nil && d.Service.ActiveState != "" {
		appendServiceDetails(t, d.Service, bold)
	}

	fmt.Fprintln(w, t.String())
	return nil
}

func appendWorkloadDetails(t *tree.Tree, d *AppStatusDetails, bold lipgloss.Style) {
	workloadNode := t.Child(bold.Render("Workload"))
	workloadNode.Child(fmt.Sprintf("Type:   %s", d.AppType))
	workloadNode.Child(fmt.Sprintf("Status: %s", statusStyle(d.Status).Render(d.Status)))
	if d.Port > 0 {
		if d.ContainerPort > 0 && d.ContainerPort != d.Port {
			workloadNode.Child(fmt.Sprintf("Port:   %d -> %d", d.Port, d.ContainerPort))
		} else {
			workloadNode.Child(fmt.Sprintf("Port:   %d", d.Port))
		}
	}
	if len(d.Domains) > 0 {
		workloadNode.Child(fmt.Sprintf("Domain: %s", strings.Join(d.Domains, ", ")))
	}
	if len(d.Tags) > 0 {
		workloadNode.Child(fmt.Sprintf("Tags:   %s", strings.Join(d.Tags, ", ")))
	}
	if d.Healthcheck != "" {
		workloadNode.Child(fmt.Sprintf("Health: %s", d.Healthcheck))
	}
}

func appendSourceDetails(t *tree.Tree, d *AppStatusDetails, bold lipgloss.Style) {
	sourceNode := t.Child(bold.Render("Source"))
	sourceNode.Child(fmt.Sprintf("Repo:   %s", d.RepoURL))
	sourceNode.Child(fmt.Sprintf("Branch: %s", d.Branch))
	sourceNode.Child(fmt.Sprintf("Commit: %s", d.Commit))
}

func appendServiceDetails(t *tree.Tree, svc *systemd.ServiceProperties, bold lipgloss.Style) {
	serviceNode := t.Child(bold.Render("Systemd Service"))
	serviceNode.Child(fmt.Sprintf("State:   %s (%s)", svc.ActiveState, svc.SubState))
	if svc.MainPID > 0 {
		serviceNode.Child(fmt.Sprintf("Main PID: %d", svc.MainPID))
	}
	if svc.ActiveEnterTimestamp != "" {
		serviceNode.Child(fmt.Sprintf("Since:    %s", svc.ActiveEnterTimestamp))
	}
	if svc.MemoryCurrent > 0 {
		serviceNode.Child(fmt.Sprintf("Memory:   %s", formatMemory(svc.MemoryCurrent)))
	}
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
