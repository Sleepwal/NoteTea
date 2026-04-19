package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name              string
	TitleFg           string
	TitleBg           string
	UserPrefixFg      string
	AssistantPrefixFg string
	UserMsgFg         string
	AssistantMsgFg    string
	ErrorFg           string
	StatusFg          string
	ModelInfoFg       string
	HelpFg            string
	InputBorderFg     string
	InputFocusFg      string
	SpinnerFg         string
	SystemMsgFg       string
	StatsFg           string
	PickerActiveFg    string
	PickerBorderFg    string
	NoteTitleFg       string
	NoteTagFg         string
	NotePreviewFg     string
}

var DarkTheme = Theme{
	Name:              "dark",
	TitleFg:           "#FFFFFF",
	TitleBg:           "#7D56F4",
	UserPrefixFg:      "#7D56F4",
	AssistantPrefixFg: "#06B6D4",
	UserMsgFg:         "#E0E0E0",
	AssistantMsgFg:    "#D0D0D0",
	ErrorFg:           "#FF6B6B",
	StatusFg:          "#888888",
	ModelInfoFg:       "#7D56F4",
	HelpFg:            "#626262",
	InputBorderFg:     "#7D56F4",
	InputFocusFg:      "#FF6B6B",
	SpinnerFg:         "#7D56F4",
	SystemMsgFg:       "#FBBF24",
	StatsFg:           "#888888",
	PickerActiveFg:    "#7D56F4",
	PickerBorderFg:    "#7D56F4",
	NoteTitleFg:       "#C4B5FD",
	NoteTagFg:         "#34D399",
	NotePreviewFg:     "#6B7280",
}

var LightTheme = Theme{
	Name:              "light",
	TitleFg:           "#FFFFFF",
	TitleBg:           "#6D28D9",
	UserPrefixFg:      "#6D28D9",
	AssistantPrefixFg: "#0891B2",
	UserMsgFg:         "#1F2937",
	AssistantMsgFg:    "#374151",
	ErrorFg:           "#DC2626",
	StatusFg:          "#6B7280",
	ModelInfoFg:       "#6D28D9",
	HelpFg:            "#9CA3AF",
	InputBorderFg:     "#6D28D9",
	InputFocusFg:      "#DC2626",
	SpinnerFg:         "#6D28D9",
	SystemMsgFg:       "#D97706",
	StatsFg:           "#6B7280",
	PickerActiveFg:    "#6D28D9",
	PickerBorderFg:    "#6D28D9",
	NoteTitleFg:       "#7C3AED",
	NoteTagFg:         "#059669",
	NotePreviewFg:     "#9CA3AF",
}

var CatppuccinTheme = Theme{
	Name:              "catppuccin",
	TitleFg:           "#CDD6F4",
	TitleBg:           "#CBA6F7",
	UserPrefixFg:      "#CBA6F7",
	AssistantPrefixFg: "#89DCEB",
	UserMsgFg:         "#CDD6F4",
	AssistantMsgFg:    "#BAC2DE",
	ErrorFg:           "#F38BA8",
	StatusFg:          "#6C7086",
	ModelInfoFg:       "#CBA6F7",
	HelpFg:            "#585B70",
	InputBorderFg:     "#CBA6F7",
	InputFocusFg:      "#F38BA8",
	SpinnerFg:         "#CBA6F7",
	SystemMsgFg:       "#F9E2AF",
	StatsFg:           "#6C7086",
	PickerActiveFg:    "#CBA6F7",
	PickerBorderFg:    "#CBA6F7",
	NoteTitleFg:       "#CBA6F7",
	NoteTagFg:         "#A6E3A1",
	NotePreviewFg:     "#6C7086",
}

var AvailableThemes = []Theme{DarkTheme, LightTheme, CatppuccinTheme}

var currentTheme = DarkTheme

func SetTheme(theme Theme) {
	currentTheme = theme
	applyTheme()
}

func GetTheme() *Theme {
	return &currentTheme
}

func SetThemeByName(name string) bool {
	for _, t := range AvailableThemes {
		if t.Name == name {
			SetTheme(t)
			return true
		}
	}
	return false
}

var (
	TitleStyle              lipgloss.Style
	UserPrefixStyle         lipgloss.Style
	AssistantPrefixStyle    lipgloss.Style
	UserMsgStyle            lipgloss.Style
	AssistantMsgStyle       lipgloss.Style
	ErrorStyle              lipgloss.Style
	StatusBarStyle          lipgloss.Style
	ModelInfoStyle          lipgloss.Style
	HelpStyle               lipgloss.Style
	InputBorderStyle        lipgloss.Style
	InputBorderFocusedStyle lipgloss.Style
	SpinnerStyle            lipgloss.Style
	SystemMsgStyle          lipgloss.Style
	StatsStyle              lipgloss.Style
	ModelPickerActiveStyle  lipgloss.Style
	ModelPickerBorderStyle  lipgloss.Style
	NoteTitleStyle          lipgloss.Style
	NoteTagStyle            lipgloss.Style
	NotePreviewStyle        lipgloss.Style
)

func applyTheme() {
	t := currentTheme
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.TitleFg)).Background(lipgloss.Color(t.TitleBg)).Padding(0, 1)
	UserPrefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.UserPrefixFg))
	AssistantPrefixStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.AssistantPrefixFg))
	UserMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.UserMsgFg))
	AssistantMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.AssistantMsgFg))
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.ErrorFg)).Bold(true)
	StatusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.StatusFg))
	ModelInfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.ModelInfoFg)).Bold(true)
	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.HelpFg))
	InputBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(t.InputBorderFg)).Padding(0, 1)
	InputBorderFocusedStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(t.InputFocusFg)).Padding(0, 1)
	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.SpinnerFg))
	SystemMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.SystemMsgFg)).Italic(true)
	StatsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.StatsFg)).Italic(true)
	ModelPickerActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.PickerActiveFg)).Bold(true)
	ModelPickerBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(t.PickerBorderFg)).Padding(1, 2)
	NoteTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.NoteTitleFg)).Bold(true)
	NoteTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.NoteTagFg))
	NotePreviewStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.NotePreviewFg)).Italic(true)
}

func init() {
	applyTheme()
}
