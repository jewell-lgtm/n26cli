package tui

import "github.com/charmbracelet/lipgloss"

// N26 brand colors.
var (
	Accent = lipgloss.AdaptiveColor{Light: "#2D8A76", Dark: "#36A18B"}
	Subtle = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#666666"}
	Error  = lipgloss.AdaptiveColor{Light: "#D32F2F", Dark: "#EF5350"}
	Green  = lipgloss.AdaptiveColor{Light: "#2E7D32", Dark: "#66BB6A"}
)

// Reusable styles.
var (
	Title = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Subtle).
			Italic(true)

	InfoBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	ErrorBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Error).
			Foreground(Error).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	SuccessMessage = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	Paragraph = lipgloss.NewStyle().
			MarginBottom(1)

	HelpText = lipgloss.NewStyle().
			Foreground(Subtle).
			Italic(true)
)
