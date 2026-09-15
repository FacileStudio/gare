package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

const (
	colorSuccess = "#10B981"
	colorError   = "#EF4444"
	colorInfo    = "#06B6D4"
	colorWarning = "#F59E0B"
	colorSubtle  = "#6B7280"
	colorPrimary = "#7D56F4"
)

// AppRow holds tabular display data for an application.
type AppRow struct {
	Name   string
	Port   int
	Domain string
	Commit string
	Status string
}

func printSuccess(msg string) {
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true).Render("✓")
	lipgloss.Println(icon, msg)
}

func printError(msg string) {
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Bold(true).Render("✗")
	lipgloss.Fprintln(os.Stderr, icon, msg)
}

func printInfo(msg string) {
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorInfo)).Bold(true).Render("▸")
	lipgloss.Println(icon, msg)
}

func printWarning(msg string) {
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning)).Bold(true).Render("!")
	lipgloss.Fprintln(os.Stderr, icon, msg)
}

func statusStyle(status string) lipgloss.Style {
	switch strings.ToLower(status) {
	case "active", "static":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true)
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorError)).Bold(true)
	case "inactive":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorSubtle))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning))
	}
}

func renderAppTable(rows []AppRow) string {
	header := lipgloss.NewStyle().Foreground(lipgloss.Color(colorPrimary)).Bold(true)
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(header).
		Headers("APP", "PORT", "DOMAIN", "COMMIT", "STATUS").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			switch col {
			case 0:
				return lipgloss.NewStyle().Foreground(lipgloss.Color(colorPrimary)).Bold(true)
			case 1:
				return lipgloss.NewStyle().Foreground(lipgloss.Color(colorInfo))
			case 4:
				return statusStyle(rows[row].Status)
			default:
				return lipgloss.NewStyle().Foreground(lipgloss.Color(colorSubtle))
			}
		})
	for _, r := range rows {
		portStr := fmt.Sprintf("%d", r.Port)
		if r.Port == 0 {
			portStr = "-"
		}
		t.Row(r.Name, portStr, r.Domain, r.Commit, r.Status)
	}
	return t.Render()
}
