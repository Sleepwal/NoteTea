package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/store"
)

func (m AppModel) handleKeyMsg(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showModelPicker {
		return m.handleModelPickerKey(message)
	}

	switch message.String() {
	case "ctrl+c":
		if m.showModelPicker {
			m.showModelPicker = false
			return m, nil
		}
		m.cleanupStream()
		if m.cancelFunc != nil {
			m.cancelFunc()
			m.cancelFunc = nil
		}
		return m, tea.Quit

	case "ctrl+enter", "alt+enter":
		if m.focused == FocusInput && !m.loading {
			return m.handleSend()
		}

	case "r":
		if m.hasError && !m.loading {
			return m.handleRetry()
		}

	case "ctrl+l":
		if !m.loading {
			m.messages = nil
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil
		}

	case "ctrl+n":
		if !m.loading {
			m.messages = nil
			m.inputHistory = nil
			m.historyIndex = -1
			m.currentInput = ""
			m.conversation = store.NewConversation(m.client.Model)
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil
		}

	case "tab":
		if m.focused == FocusInput {
			m.focused = FocusChat
			m.textarea.Blur()
		} else {
			m.focused = FocusInput
			m.textarea.Focus()
		}
		return m, nil

	case "ctrl+m":
		if !m.loading {
			m.showModelPicker = !m.showModelPicker
			if m.showModelPicker {
				m.modelCursor = 0
				for i, model := range api.AvailableModels {
					if model == m.client.Model {
						m.modelCursor = i
						break
					}
				}
			}
			return m, nil
		}

	case "esc":
		if m.showModelPicker {
			m.showModelPicker = false
			return m, nil
		}
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.loading && m.cancelFunc != nil {
			m.cancelFunc()
			m.cancelFunc = nil
			m.loading = false
			m.cleanupStream()
			if len(m.messages) > 0 {
				last := &m.messages[len(m.messages)-1]
				if last.Streaming {
					last.Streaming = false
				}
			}
			m.viewport.SetContent(m.renderMessages())
			return m, nil
		}
		return m, nil

	case "up":
		if m.focused == FocusInput && len(m.inputHistory) > 0 {
			if m.historyIndex == -1 {
				m.currentInput = m.textarea.Value()
				m.historyIndex = len(m.inputHistory) - 1
			} else if m.historyIndex > 0 {
				m.historyIndex--
			}
			m.textarea.SetValue(m.inputHistory[m.historyIndex])
			return m, nil
		}

	case "down":
		if m.focused == FocusInput && m.historyIndex != -1 {
			if m.historyIndex < len(m.inputHistory)-1 {
				m.historyIndex++
				m.textarea.SetValue(m.inputHistory[m.historyIndex])
			} else {
				m.historyIndex = -1
				m.textarea.SetValue(m.currentInput)
			}
			return m, nil
		}

	case "?":
		if !m.loading && m.focused == FocusChat {
			m.showHelp = !m.showHelp
			return m, nil
		}
	}

	return m.delegateToComponents(message)
}

func (m AppModel) delegateToComponents(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.textarea, cmd = m.textarea.Update(teaMsg)
	cmds = append(cmds, cmd)

	if m.focused == FocusChat {
		m.viewport, cmd = m.viewport.Update(teaMsg)
		cmds = append(cmds, cmd)
	}

	m.spinner, cmd = m.spinner.Update(teaMsg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m AppModel) handleModelPickerKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "up", "k":
		if m.modelCursor > 0 {
			m.modelCursor--
		}
	case "down", "j":
		if m.modelCursor < len(api.AvailableModels)-1 {
			m.modelCursor++
		}
	case "enter":
		m.client.Model = api.AvailableModels[m.modelCursor]
		m.showModelPicker = false
	case "esc", "ctrl+c":
		m.showModelPicker = false
	}
	return m, nil
}
