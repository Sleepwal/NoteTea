package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/agenttea/internal/logger"
	"github.com/user/agenttea/internal/store"
	"github.com/user/agenttea/internal/ui"
)

const quizSystemPrompt = `你是一个知识巩固助手。用户会提供一份学习笔记，请你根据笔记内容逐个提出问题来帮助用户巩固知识点。

规则：
1. 每次只提一个问题
2. 问题应该覆盖笔记中的核心概念和关键细节
3. 等用户回答后再评价其回答
4. 如果回答正确，给予肯定并提下一个问题
5. 如果回答不完整或有误，温和地指出并补充正确信息，然后提下一个问题
6. 当所有重要知识点都已覆盖后，给出整体评价和学习建议
7. 用中文交流`

func (m AppModel) handleNotePickerKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := message.String()

	switch key {
	case "up", "k":
		m.noteDeleteConfirm = false
		if m.noteCursor > 0 {
			m.noteCursor--
		}
	case "down", "j":
		m.noteDeleteConfirm = false
		if m.noteCursor < len(m.noteList)-1 {
			m.noteCursor++
		}
	case "enter":
		m.noteDeleteConfirm = false
		if len(m.noteList) > 0 {
			selected := m.noteList[m.noteCursor]
			note, err := store.LoadNote(selected.ID)
			if err != nil || note == nil {
				return m, nil
			}
			m.currentNote = note
			m.noteEditorMode = "view"
			m.showNoteEditor = true
			m.showNotePicker = false
			m.initNoteViewer()
		}
	case "n":
		m.noteDeleteConfirm = false
		m.currentNote = store.NewNote("")
		m.noteEditorMode = "create"
		m.showNoteEditor = true
		m.showNotePicker = false
		m.noteTitleInput.SetValue("")
		m.noteTitleInput.Focus()
		m.noteContentInput.SetValue("")
		m.noteTagsInput.SetValue("")
		m.recalcNoteEditorLayout()
		return m, textarea.Blink
	case "e":
		m.noteDeleteConfirm = false
		if len(m.noteList) > 0 {
			selected := m.noteList[m.noteCursor]
			note, err := store.LoadNote(selected.ID)
			if err != nil || note == nil {
				return m, nil
			}
			m.currentNote = note
			m.noteEditorMode = "edit"
			m.showNoteEditor = true
			m.showNotePicker = false
			m.noteTitleInput.SetValue(note.Title)
			m.noteTitleInput.Focus()
			m.noteContentInput.SetValue(note.Content)
			m.noteTagsInput.SetValue(strings.Join(note.Tags, ", "))
			m.recalcNoteEditorLayout()
			return m, textarea.Blink
		}
	case "d":
		if len(m.noteList) > 0 {
			if m.noteDeleteConfirm {
				selected := m.noteList[m.noteCursor]
				store.DeleteNote(selected.ID)
				m.noteList = append(m.noteList[:m.noteCursor], m.noteList[m.noteCursor+1:]...)
				if m.noteCursor >= len(m.noteList) {
					m.noteCursor = len(m.noteList) - 1
				}
				m.noteDeleteConfirm = false
				if len(m.noteList) == 0 {
					m.showNotePicker = false
				}
			} else {
				m.noteDeleteConfirm = true
			}
		}
	case "q":
		m.noteDeleteConfirm = false
		if len(m.noteList) > 0 {
			return m.startQuizFromNote(&m.noteList[m.noteCursor])
		}
	case "esc", "ctrl+c":
		m.noteDeleteConfirm = false
		m.showNotePicker = false
	default:
		m.noteDeleteConfirm = false
	}
	return m, nil
}

func (m AppModel) handleNoteEditorKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+s":
		title := strings.TrimSpace(m.noteTitleInput.Value())
		content := m.noteContentInput.Value()
		tagsStr := strings.TrimSpace(m.noteTagsInput.Value())

		if title == "" {
			title = "未命名笔记"
		}

		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		if m.noteEditorMode == "create" {
			note := store.NewNote(title)
			note.Content = content
			note.Tags = tags
			if err := store.SaveNote(note); err != nil {
				logger.Error("保存笔记失败: %v", err)
			}
			m.currentNote = note
		} else if m.noteEditorMode == "edit" && m.currentNote != nil {
			m.currentNote.Title = title
			m.currentNote.Content = content
			m.currentNote.Tags = tags
			if err := store.SaveNote(m.currentNote); err != nil {
				logger.Error("保存笔记失败: %v", err)
			}
		}

		m.showNoteEditor = false
		m.noteEditorMode = ""
		m.noteTitleInput.Blur()
		m.noteContentInput.Blur()
		m.noteTagsInput.Blur()

		notes, _ := store.ListNotes()
		m.noteList = notes
		m.showNotePicker = true
		m.noteCursor = 0
		for i, n := range notes {
			if m.currentNote != nil && n.ID == m.currentNote.ID {
				m.noteCursor = i
				break
			}
		}

		return m, nil
	case "esc":
		m.showNoteEditor = false
		m.noteEditorMode = ""
		m.noteTitleInput.Blur()
		m.noteContentInput.Blur()
		m.noteTagsInput.Blur()

		notes, _ := store.ListNotes()
		m.noteList = notes
		m.showNotePicker = true
		return m, nil
	case "tab":
		if m.noteTitleInput.Focused() {
			m.noteTitleInput.Blur()
			m.noteContentInput.Focus()
			return m, textarea.Blink
		} else if m.noteContentInput.Focused() {
			m.noteContentInput.Blur()
			m.noteTagsInput.Focus()
			return m, textarea.Blink
		} else {
			m.noteTagsInput.Blur()
			m.noteTitleInput.Focus()
			return m, textarea.Blink
		}
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.noteTitleInput.Focused() {
		m.noteTitleInput, cmd = m.noteTitleInput.Update(message)
		cmds = append(cmds, cmd)
	} else if m.noteContentInput.Focused() {
		m.noteContentInput, cmd = m.noteContentInput.Update(message)
		cmds = append(cmds, cmd)
	} else if m.noteTagsInput.Focused() {
		m.noteTagsInput, cmd = m.noteTagsInput.Update(message)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) handleNoteViewerKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		m.showNoteEditor = false
		m.noteEditorMode = ""
		m.showNotePicker = true
		return m, nil
	case "e":
		if m.currentNote != nil {
			m.noteEditorMode = "edit"
			m.noteTitleInput.SetValue(m.currentNote.Title)
			m.noteTitleInput.Focus()
			m.noteContentInput.SetValue(m.currentNote.Content)
			m.noteTagsInput.SetValue(strings.Join(m.currentNote.Tags, ", "))
			m.recalcNoteEditorLayout()
			return m, textarea.Blink
		}
		return m, nil
	case "q":
		if m.currentNote != nil {
			return m.startQuizFromNote(m.currentNote)
		}
	case "up", "k":
		m.noteViewer.LineUp(1)
		return m, nil
	case "down", "j":
		m.noteViewer.LineDown(1)
		return m, nil
	case "pgup":
		m.noteViewer.HalfViewUp()
		return m, nil
	case "pgdown":
		m.noteViewer.HalfViewDown()
		return m, nil
	}
	return m, nil
}

func (m *AppModel) initNoteViewer() {
	if m.currentNote == nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", m.currentNote.Title))

	if len(m.currentNote.Tags) > 0 {
		var tagParts []string
		for _, t := range m.currentNote.Tags {
			tagParts = append(tagParts, fmt.Sprintf("#%s", t))
		}
		sb.WriteString(strings.Join(tagParts, " ") + "\n\n")
	}

	sb.WriteString(m.currentNote.Content)

	charCount := len([]rune(m.currentNote.Content))
	lineCount := strings.Count(m.currentNote.Content, "\n") + 1
	sb.WriteString(fmt.Sprintf("\n\n---\n%d 字 | %d 行 | 创建: %s | 更新: %s",
		charCount, lineCount,
		m.currentNote.CreatedAt.Format("2006-01-02 15:04"),
		m.currentNote.UpdatedAt.Format("2006-01-02 15:04"),
	))

	rendered := ui.RenderMarkdown(sb.String())
	m.noteViewer.SetContent(rendered)
	m.noteViewer.GotoTop()

	if m.width > 0 && m.height > 0 {
		m.noteViewer.Width = m.width - 8
		viewerHeight := m.height - 10
		if viewerHeight < 5 {
			viewerHeight = 5
		}
		m.noteViewer.Height = viewerHeight
	}
}

func (m AppModel) startQuizFromNote(note *store.Note) (tea.Model, tea.Cmd) {
	m.showNotePicker = false
	m.showNoteEditor = false
	m.noteEditorMode = ""
	m.noteDeleteConfirm = false
	m.noteTitleInput.Blur()
	m.noteContentInput.Blur()
	m.noteTagsInput.Blur()

	m.messages = nil
	m.inputHistory = nil
	m.historyIndex = -1
	m.currentInput = ""
	m.hasError = false
	m.conversation = store.NewConversation(m.client.Model)
	m.systemPrompt = quizSystemPrompt

	userContent := fmt.Sprintf("请根据以下笔记内容向我提问，帮助我巩固知识点：\n\n## %s\n\n%s", note.Title, note.Content)

	userMsg := ChatMessage{
		Role:      "user",
		Content:   userContent,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	assistantMsg := ChatMessage{
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
		Streaming: true,
	}
	m.messages = append(m.messages, assistantMsg)

	m.apiMessages = m.buildAPIMessages()
	m.loading = true
	m.saveConversation()
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()

	logger.Info("启动知识巩固模式, 笔记: %s", note.Title)

	return m, m.startChatRequest()
}

func (m AppModel) openNotePicker() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	notes, err := store.ListNotes()
	if err != nil {
		logger.Error("加载笔记列表失败: %v", err)
		return m, nil
	}
	m.showNotePicker = true
	m.noteList = notes
	m.noteCursor = 0
	m.noteDeleteConfirm = false
	return m, nil
}

func (m AppModel) delegateNoteEditorComponents(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.noteTitleInput, cmd = m.noteTitleInput.Update(teaMsg)
	cmds = append(cmds, cmd)

	m.noteContentInput, cmd = m.noteContentInput.Update(teaMsg)
	cmds = append(cmds, cmd)

	m.noteTagsInput, cmd = m.noteTagsInput.Update(teaMsg)
	cmds = append(cmds, cmd)

	m.spinner, cmd = m.spinner.Update(teaMsg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m AppModel) handleNoteEditorMouse(message tea.MouseMsg) (tea.Model, tea.Cmd) {
	if message.Button == tea.MouseButtonWheelUp {
		m.noteContentInput, _ = m.noteContentInput.Update(tea.KeyMsg{Type: tea.KeyUp})
		return m, nil
	}
	if message.Button == tea.MouseButtonWheelDown {
		m.noteContentInput, _ = m.noteContentInput.Update(tea.KeyMsg{Type: tea.KeyDown})
		return m, nil
	}
	return m.delegateNoteEditorComponents(message)
}

func (m AppModel) handleNoteViewerMouse(message tea.MouseMsg) (tea.Model, tea.Cmd) {
	if message.Button == tea.MouseButtonWheelUp {
		m.noteViewer.LineUp(3)
		return m, nil
	}
	if message.Button == tea.MouseButtonWheelDown {
		m.noteViewer.LineDown(3)
		return m, nil
	}
	return m, nil
}

func (m AppModel) recalcNoteEditorLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	m.noteTitleInput.SetWidth(m.width - 6)
	m.noteContentInput.SetWidth(m.width - 6)
	m.noteTagsInput.SetWidth(m.width - 6)
	m.noteViewer.Width = m.width - 8

	titleH := lipgloss.Height(m.noteTitleInput.View())
	tagsH := lipgloss.Height(m.noteTagsInput.View())
	nonContentLines := 2 + 1 + titleH + 1 + 2 + 1 + tagsH + 1 + 1
	contentH := m.height - nonContentLines
	if contentH < 5 {
		contentH = 5
	}
	m.noteContentInput.SetHeight(contentH)

	viewerHeight := m.height - 10
	if viewerHeight < 5 {
		viewerHeight = 5
	}
	m.noteViewer.Height = viewerHeight
}
