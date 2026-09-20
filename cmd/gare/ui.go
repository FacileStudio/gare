package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	colorSuccess = "#10B981"
	colorError   = "#EF4444"
	colorInfo    = "#06B6D4"
	colorWarning = "#F59E0B"
	colorSubtle  = "#6B7280"
	colorPrimary = "#7D56F4"
)

// applyColorPreference disables color for the rest of the process by exporting
// NO_COLOR, which lipgloss reads on every render.
func applyColorPreference(noColor bool) error {
	if !noColor {
		return nil
	}
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		return fmt.Errorf("disable colors: %w", err)
	}
	return nil
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
