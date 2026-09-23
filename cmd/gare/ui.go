package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// printVerbose writes an extra detail line when the command was given --verbose.
func printVerbose(ctx context.Context, format string, args ...any) {
	if !settingsFrom(ctx).verbose {
		return
	}
	icon := lipgloss.NewStyle().Foreground(lipgloss.Color(colorSubtle)).Render("·")
	lipgloss.Println(icon, fmt.Sprintf(format, args...))
}

// writeJSON renders v as an indented JSON document to w.
func writeJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
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
