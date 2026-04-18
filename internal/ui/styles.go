package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	UserPrefixStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	AssistantPrefixStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#06B6D4"))

	UserMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	AssistantMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D0D0D0"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(0, 1)

	InputBorderFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF6B6B")).
				Padding(0, 1)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))

	SystemMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).Italic(true)
)
