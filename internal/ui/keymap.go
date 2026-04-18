package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit         key.Binding
	Send         key.Binding
	ClearHistory key.Binding
	NewChat      key.Binding
	Up           key.Binding
	Down         key.Binding
	Tab          key.Binding
	Help         key.Binding
	Cancel       key.Binding
}

var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "退出"),
	),
	Send: key.NewBinding(
		key.WithKeys("ctrl+enter", "alt+enter"),
		key.WithHelp("ctrl+enter/alt+enter", "发送"),
	),
	ClearHistory: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "清空历史"),
	),
	NewChat: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "新对话"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "上滚"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "下滚"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "切换焦点"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "帮助"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "取消请求"),
	),
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Send, k.Tab, k.Help, k.Quit,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.ClearHistory, k.NewChat},
		{k.Up, k.Down, k.Tab},
		{k.Help, k.Cancel, k.Quit},
	}
}
