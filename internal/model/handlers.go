package model

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/store"
	"github.com/user/agenttea/internal/ui"
)

func (m AppModel) handleKeyMsg(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showModelPicker {
		return m.handleModelPickerKey(message)
	}
	if m.showConvPicker {
		return m.handleConvPickerKey(message)
	}
	if m.showPromptPicker {
		return m.handlePromptPickerKey(message)
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

	case "ctrl+p":
		if !m.loading {
			convs, err := store.ListConversations()
			if err != nil || len(convs) == 0 {
				return m, nil
			}
			m.showConvPicker = true
			m.convList = convs
			m.convCursor = 0
			for i, c := range convs {
				if m.conversation != nil && c.ID == m.conversation.ID {
					m.convCursor = i
					break
				}
			}
			return m, nil
		}

	case "ctrl+e":
		if !m.loading && len(m.messages) > 0 {
			m.exportConversation()
			return m, nil
		}

	case "ctrl+s":
		if !m.loading {
			m.showPromptPicker = true
			m.promptCursor = 0
			return m, nil
		}

	case "ctrl+y":
		if !m.loading && m.focused == FocusChat {
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "assistant" && !m.messages[i].Streaming {
					ok, preview := ui.CopyLastCodeBlock(m.messages[i].Content)
					if ok {
						m.messages = append(m.messages, ChatMessage{
							Role:      "system",
							Content:   fmt.Sprintf("已复制代码块: %s", preview),
							Timestamp: time.Now(),
						})
						m.viewport.SetContent(m.renderMessages())
						m.viewport.GotoBottom()
					} else {
						m.messages = append(m.messages, ChatMessage{
							Role:      "system",
							Content:   "未找到可复制的代码块",
							Timestamp: time.Now(),
						})
						m.viewport.SetContent(m.renderMessages())
						m.viewport.GotoBottom()
					}
					return m, nil
				}
			}
		}

	case "ctrl+t":
		if !m.loading {
			themes := ui.AvailableThemes
			current := ui.GetTheme()
			nextIdx := 0
			for i, t := range themes {
				if t.Name == current.Name {
					nextIdx = (i + 1) % len(themes)
					break
				}
			}
			ui.SetTheme(themes[nextIdx])
			m.messages = append(m.messages, ChatMessage{
				Role:      "system",
				Content:   fmt.Sprintf("主题已切换为: %s", themes[nextIdx].Name),
				Timestamp: time.Now(),
			})
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil
		}

	case "esc":
		if m.showPromptPicker {
			m.showPromptPicker = false
			return m, nil
		}
		if m.showConvPicker {
			m.showConvPicker = false
			return m, nil
		}
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

func (m AppModel) handleConvPickerKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "up", "k":
		if m.convCursor > 0 {
			m.convCursor--
		}
	case "down", "j":
		if m.convCursor < len(m.convList)-1 {
			m.convCursor++
		}
	case "enter":
		selected := m.convList[m.convCursor]
		conv, err := store.LoadConversation(selected.ID)
		if err != nil || conv == nil {
			m.showConvPicker = false
			return m, nil
		}
		m.conversation = conv
		restoreMsgs := make([]ChatMessage, 0, len(conv.Messages))
		for _, sm := range conv.Messages {
			restoreMsgs = append(restoreMsgs, ChatMessage{
				Role:      sm.Role,
				Content:   sm.Content,
				Timestamp: sm.Timestamp,
				Stats:     sm.Stats,
			})
		}
		m.messages = restoreMsgs
		m.inputHistory = nil
		m.historyIndex = -1
		m.currentInput = ""
		m.hasError = false
		m.showConvPicker = false
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
	case "d":
		if len(m.convList) > 0 {
			selected := m.convList[m.convCursor]
			store.DeleteConversation(selected.ID)
			m.convList = append(m.convList[:m.convCursor], m.convList[m.convCursor+1:]...)
			if m.convCursor >= len(m.convList) {
				m.convCursor = len(m.convList) - 1
			}
			if len(m.convList) == 0 {
				m.showConvPicker = false
			}
		}
	case "esc", "ctrl+c":
		m.showConvPicker = false
	}
	return m, nil
}

func (m AppModel) handlePromptPickerKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "up", "k":
		if m.promptCursor > 0 {
			m.promptCursor--
		}
	case "down", "j":
		if m.promptCursor < len(m.promptPresets)-1 {
			m.promptCursor++
		}
	case "enter":
		if len(m.promptPresets) > 0 {
			m.systemPrompt = m.promptPresets[m.promptCursor].Prompt
		}
		m.showPromptPicker = false
	case "c":
		m.systemPrompt = ""
		m.showPromptPicker = false
	case "esc", "ctrl+c":
		m.showPromptPicker = false
	}
	return m, nil
}
