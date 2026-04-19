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

// quizSystemPrompt 是 AI 知识巩固模式的专用 System Prompt。
// 指导 AI 根据用户提供的笔记内容逐个提问，帮助巩固知识点。
const quizSystemPrompt = `你是一个知识巩固助手。用户会提供一份学习笔记，请你根据笔记内容逐个提出问题来帮助用户巩固知识点。

规则：
1. 每次只提一个问题
2. 问题应该覆盖笔记中的核心概念和关键细节
3. 等用户回答后再评价其回答
4. 如果回答正确，给予肯定并提下一个问题
5. 如果回答不完整或有误，温和地指出并补充正确信息，然后提下一个问题
6. 当所有重要知识点都已覆盖后，给出整体评价和学习建议
7. 用中文交流`

// handleNotePickerKey 处理笔记列表弹窗中的键盘事件。
//
// 快捷键:
//   - ↑/k, ↓/j: 上下导航
//   - Enter: 查看选中笔记（切换到 view 模式）
//   - n: 新建笔记（切换到编辑器 create 模式）
//   - e: 编辑选中笔记（切换到编辑器 edit 模式）
//   - d: 删除笔记（需按两次确认）
//   - q: 启动 AI 知识巩固模式
//   - Esc/Ctrl+C: 关闭笔记列表
//
// 删除确认机制：首次按 d 设置 noteDeleteConfirm=true 并显示确认提示，
// 再次按 d 执行删除，按其他任何键取消确认。
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
				// 二次确认：执行删除
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
				// 首次按 d：进入确认状态
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
		// 其他按键重置删除确认状态
		m.noteDeleteConfirm = false
	}
	return m, nil
}

// handleNoteEditorKey 处理笔记编辑器中的键盘事件。
//
// 快捷键:
//   - Ctrl+S: 保存笔记（新建或更新）
//   - Esc: 取消编辑，返回笔记列表
//   - Tab: 在标题→内容→标签三个输入框间循环切换焦点
//
// 保存逻辑：
//   - 标题为空时自动设为"未命名笔记"
//   - 标签从逗号分隔的字符串解析为 []string
//   - 保存后刷新笔记列表并定位到刚保存的笔记
func (m AppModel) handleNoteEditorKey(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+s":
		title := strings.TrimSpace(m.noteTitleInput.Value())
		content := m.noteContentInput.Value()
		tagsStr := strings.TrimSpace(m.noteTagsInput.Value())

		// 标题为空时使用默认标题
		if title == "" {
			title = "未命名笔记"
		}

		// 解析逗号分隔的标签字符串
		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		// 根据编辑模式执行新建或更新
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

		// 关闭编辑器，返回笔记列表
		m.showNoteEditor = false
		m.noteEditorMode = ""
		m.noteTitleInput.Blur()
		m.noteContentInput.Blur()
		m.noteTagsInput.Blur()

		// 刷新列表并定位到刚保存的笔记
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
		// 取消编辑，返回笔记列表
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
		// 在标题→内容→标签间循环切换焦点
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

	// 将普通按键事件传递给当前聚焦的 textarea 组件
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

// handleNoteViewerKey 处理笔记查看模式中的键盘事件。
//
// 快捷键:
//   - Esc: 返回笔记列表
//   - e: 切换到编辑模式
//   - q: 启动 AI 知识巩固
//   - ↑/k, ↓/j: 逐行滚动
//   - PgUp/PgDn: 半页滚动
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

// initNoteViewer 初始化笔记查看器的 viewport。
// 将笔记内容（标题 + 标签 + 正文 + 统计信息）渲染为 Markdown，
// 设置到 noteViewer 中并滚动到顶部。
func (m *AppModel) initNoteViewer() {
	if m.currentNote == nil {
		return
	}
	var sb strings.Builder
	// 标题
	sb.WriteString(fmt.Sprintf("# %s\n\n", m.currentNote.Title))

	// 标签（#tag1 #tag2 格式）
	if len(m.currentNote.Tags) > 0 {
		var tagParts []string
		for _, t := range m.currentNote.Tags {
			tagParts = append(tagParts, fmt.Sprintf("#%s", t))
		}
		sb.WriteString(strings.Join(tagParts, " ") + "\n\n")
	}

	// 正文
	sb.WriteString(m.currentNote.Content)

	// 统计信息：字数、行数、创建/更新时间
	charCount := len([]rune(m.currentNote.Content))
	lineCount := strings.Count(m.currentNote.Content, "\n") + 1
	sb.WriteString(fmt.Sprintf("\n\n---\n%d 字 | %d 行 | 创建: %s | 更新: %s",
		charCount, lineCount,
		m.currentNote.CreatedAt.Format("2006-01-02 15:04"),
		m.currentNote.UpdatedAt.Format("2006-01-02 15:04"),
	))

	// 渲染 Markdown 并设置到 viewport
	rendered := ui.RenderMarkdown(sb.String())
	m.noteViewer.SetContent(rendered)
	m.noteViewer.GotoTop()

	// 根据当前终端尺寸调整 viewport 大小
	if m.width > 0 && m.height > 0 {
		m.noteViewer.Width = m.width - 8
		viewerHeight := m.height - 10
		if viewerHeight < 5 {
			viewerHeight = 5
		}
		m.noteViewer.Height = viewerHeight
	}
}

// startQuizFromNote 启动 AI 知识巩固模式。
// 该方法复用现有聊天架构：
//  1. 关闭所有笔记弹窗
//  2. 创建新对话，设置专用的 quizSystemPrompt
//  3. 将笔记内容作为首条用户消息发送
//  4. 触发 AI 流式请求
//
// 整个问答过程就是一场特殊引导的对话，完全复用流式渲染、消息保存等基础设施。
func (m AppModel) startQuizFromNote(note *store.Note) (tea.Model, tea.Cmd) {
	// 关闭所有笔记弹窗
	m.showNotePicker = false
	m.showNoteEditor = false
	m.noteEditorMode = ""
	m.noteDeleteConfirm = false
	m.noteTitleInput.Blur()
	m.noteContentInput.Blur()
	m.noteTagsInput.Blur()

	// 重置对话状态
	m.messages = nil
	m.inputHistory = nil
	m.historyIndex = -1
	m.currentInput = ""
	m.hasError = false
	m.conversation = store.NewConversation(m.client.Model)
	m.systemPrompt = quizSystemPrompt

	// 构造用户消息：将笔记内容嵌入请求
	userContent := fmt.Sprintf("请根据以下笔记内容向我提问，帮助我巩固知识点：\n\n## %s\n\n%s", note.Title, note.Content)

	userMsg := ChatMessage{
		Role:      "user",
		Content:   userContent,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	// 添加空的 assistant 消息占位，用于流式填充
	assistantMsg := ChatMessage{
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
		Streaming: true,
	}
	m.messages = append(m.messages, assistantMsg)

	// 构建 API 消息并启动流式请求
	m.apiMessages = m.buildAPIMessages()
	m.loading = true
	m.saveConversation()
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()

	logger.Info("启动知识巩固模式, 笔记: %s", note.Title)

	return m, m.startChatRequest()
}

// openNotePicker 打开笔记列表弹窗。
// 若 AI 正在流式响应中则忽略（避免状态冲突）。
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

// delegateNoteEditorComponents 将 tea.Msg 分发给笔记编辑器的所有子组件。
// 包括标题输入框、内容输入框、标签输入框和加载动画。
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

// handleNoteEditorMouse 处理笔记编辑器中的鼠标事件。
// 滚轮上下滚动通过向 textarea 发送 ↑/↓ 键事件实现
// （bubbles/textarea 不直接支持鼠标滚轮）。
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

// handleNoteViewerMouse 处理笔记查看器中的鼠标事件。
// 使用 viewport 的 LineUp/LineDown 实现滚轮滚动。
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

// recalcNoteEditorLayout 在终端窗口大小变化时重新计算笔记编辑器的布局。
// 根据当前终端尺寸调整各输入框的宽度，以及内容输入框的高度
// （填满标题、标签、帮助文本之外的剩余空间）。
func (m AppModel) recalcNoteEditorLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// 设置各输入框宽度（终端宽度减去 Padding 和边框）
	m.noteTitleInput.SetWidth(m.width - 6)
	m.noteContentInput.SetWidth(m.width - 6)
	m.noteTagsInput.SetWidth(m.width - 6)
	m.noteViewer.Width = m.width - 8

	// 动态计算内容输入框高度
	// 布局各行：标题栏 + "标题:" + titleInput + "内容:" + contentInput + "标签:" + tagsInput + 帮助文本 + Padding
	titleH := lipgloss.Height(m.noteTitleInput.View())
	tagsH := lipgloss.Height(m.noteTagsInput.View())
	nonContentLines := 2 + 1 + titleH + 1 + 2 + 1 + tagsH + 1 + 1
	contentH := m.height - nonContentLines
	if contentH < 5 {
		contentH = 5
	}
	m.noteContentInput.SetHeight(contentH)

	// 查看器 viewport 高度
	viewerHeight := m.height - 10
	if viewerHeight < 5 {
		viewerHeight = 5
	}
	m.noteViewer.Height = viewerHeight
}
