package model

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/config"
	"github.com/user/agenttea/internal/logger"
	"github.com/user/agenttea/internal/msg"
	"github.com/user/agenttea/internal/plugin"
	"github.com/user/agenttea/internal/store"
	"github.com/user/agenttea/internal/ui"
)

type FocusArea int

const (
	FocusInput FocusArea = iota
	FocusChat
)

type ChatMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
	Streaming bool
	Stats     string
}

type AppModel struct {
	width  int
	height int

	focused          FocusArea
	loading          bool
	showHelp         bool
	hasError         bool
	showModelPicker  bool
	modelCursor      int
	showConvPicker   bool
	convCursor       int
	convList         []store.Conversation
	systemPrompt     string
	showPromptPicker bool
	promptCursor     int
	promptPresets    []config.SystemPromptPreset

	// 笔记功能相关状态
	showNotePicker    bool           // 是否显示笔记列表弹窗
	noteCursor        int            // 笔记列表中的光标位置
	noteList          []store.Note   // 笔记列表数据
	showNoteEditor    bool           // 是否显示笔记编辑器/查看器
	noteEditorMode    string         // 编辑器模式: "create" | "edit" | "view"
	currentNote       *store.Note    // 当前操作的笔记（编辑/查看）
	noteTitleInput    textarea.Model // 笔记标题输入框
	noteContentInput  textarea.Model // 笔记内容输入框（Markdown）
	noteTagsInput     textarea.Model // 笔记标签输入框（逗号分隔）
	noteDeleteConfirm bool           // 删除确认状态（需按两次 d 确认）
	noteViewer        viewport.Model // 笔记查看器的 viewport（支持滚动）

	messages     []ChatMessage
	inputHistory []string
	historyIndex int
	currentInput string

	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	client       *api.Client
	streamReader io.ReadCloser
	streamScan   *bufio.Scanner
	cancelFunc   context.CancelFunc

	apiMessages []api.Message

	conversation *store.Conversation

	renderedPrefix string
	hookManager    *plugin.Manager
}

func NewAppModel(client *api.Client) AppModel {
	ta := textarea.New()
	ta.Placeholder = "在此输入消息..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = ui.SpinnerStyle

	vp := viewport.New(80, 20)

	noteTitleTa := textarea.New()
	noteTitleTa.Placeholder = "输入笔记标题..."
	noteTitleTa.Prompt = "┃ "
	noteTitleTa.CharLimit = 0
	noteTitleTa.SetHeight(1)
	noteTitleTa.ShowLineNumbers = false

	noteContentTa := textarea.New()
	noteContentTa.Placeholder = "输入笔记内容（Markdown 格式）..."
	noteContentTa.Prompt = "┃ "
	noteContentTa.CharLimit = 0
	noteContentTa.SetHeight(8)
	noteContentTa.ShowLineNumbers = false

	noteTagsTa := textarea.New()
	noteTagsTa.Placeholder = "标签（逗号分隔，如: go, 并发, 笔记）"
	noteTagsTa.Prompt = "┃ "
	noteTagsTa.CharLimit = 0
	noteTagsTa.SetHeight(1)
	noteTagsTa.ShowLineNumbers = false

	noteVp := viewport.New(80, 15)

	conv := store.NewConversation(client.Model)
	if lastConv, err := store.LoadLastConversation(); err == nil && lastConv != nil {
		conv = lastConv
		logger.Info("恢复上次对话: %s", conv.ID)
	}

	restoreMsgs := make([]ChatMessage, 0, len(conv.Messages))
	for _, sm := range conv.Messages {
		restoreMsgs = append(restoreMsgs, ChatMessage{
			Role:      sm.Role,
			Content:   sm.Content,
			Timestamp: sm.Timestamp,
			Stats:     sm.Stats,
		})
	}

	cfg, _ := config.Load()
	var sysPrompt string
	var presets []config.SystemPromptPreset
	if cfg != nil {
		sysPrompt = cfg.SystemPrompt
		presets = cfg.PromptPresets
		if cfg.Theme != "" {
			ui.SetThemeByName(cfg.Theme)
		}
	}
	if len(presets) == 0 {
		presets = config.DefaultConfig.PromptPresets
	}

	return AppModel{
		focused:          FocusInput,
		loading:          false,
		showHelp:         false,
		messages:         restoreMsgs,
		inputHistory:     []string{},
		historyIndex:     -1,
		textarea:         ta,
		viewport:         vp,
		spinner:          sp,
		client:           client,
		conversation:     conv,
		systemPrompt:     sysPrompt,
		promptPresets:    presets,
		hookManager:      initHookManager(cfg),
		noteTitleInput:   noteTitleTa,
		noteContentInput: noteContentTa,
		noteTagsInput:    noteTagsTa,
		noteViewer:       noteVp,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func initHookManager(cfg *config.Config) *plugin.Manager {
	mgr := plugin.NewManager()
	if cfg != nil {
		for _, hc := range cfg.Hooks {
			if hc.Enabled {
				mgr.AddHook(plugin.HookType(hc.Type), hc.Command)
			}
		}
	}
	return mgr
}

func (m AppModel) Update(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := teaMsg.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.recalcLayout()
		m.recalcNoteEditorLayout()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyMsg:
		model, cmd := m.handleKeyMsg(message)
		result := model.(AppModel)
		result.syncNoteEditorHeight()
		return result, cmd

	case tea.MouseMsg:
		if m.showNoteEditor && m.noteEditorMode != "view" {
			model, cmd := m.handleNoteEditorMouse(message)
			result := model.(*AppModel)
			result.syncNoteEditorHeight()
			return result, cmd
		}
		if m.showNoteEditor && m.noteEditorMode == "view" {
			model, cmd := m.handleNoteViewerMouse(message)
			result := model.(AppModel)
			result.syncNoteEditorHeight()
			return result, cmd
		}

	case msg.StreamStartMsg:
		m.streamReader = message.Reader
		m.cancelFunc = message.CancelCtx
		m.streamScan = bufio.NewScanner(m.streamReader)
		m.streamScan.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		m.renderedPrefix = ""
		return m, m.readNextStreamToken()

	case msg.StreamTokenMsg:
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			if last.Streaming {
				last.Content += message.Content
				if m.renderedPrefix == "" {
					m.renderedPrefix = m.renderMessagesExceptLast()
				}
				prefix := ui.AssistantPrefixStyle.Render("[Assistant]")
				content := ui.AssistantMsgStyle.Render(last.Content)
				m.viewport.SetContent(m.renderedPrefix + fmt.Sprintf("%s %s\n", prefix, content))
				m.viewport.GotoBottom()
			}
		}
		return m, m.readNextStreamToken()

	case msg.StreamDoneMsg:
		m.handleStreamDone(message)
		return m, nil

	case msg.ApiErrorMsg:
		m.handleApiError(message)
		return m, nil
	}

	model, cmd := m.delegateToComponents(teaMsg)
	result := model.(AppModel)
	result.syncNoteEditorHeight()
	return result, cmd
}

// syncNoteEditorHeight 在 Update 的返回路径中同步笔记编辑器内容输入框的高度。
// 由于 AppModel 是值类型，在值接收者方法（如 renderNoteEditor）中调用 SetHeight
// 修改的是副本，不会持久化到 Bubble Tea 的实际模型状态。
// 因此必须在 Update 方法中通过指针接收者设置高度，确保修改能传回框架。
func (m *AppModel) syncNoteEditorHeight() {
	if !m.showNoteEditor || m.noteEditorMode == "view" || m.height == 0 {
		return
	}
	titleH := lipgloss.Height(m.noteTitleInput.View())
	tagsH := lipgloss.Height(m.noteTagsInput.View())
	contentH := m.height - 9 - titleH - tagsH
	if contentH < 5 {
		contentH = 5
	}
	m.noteContentInput.SetHeight(contentH)
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	titleBar := ui.TitleStyle.Render(fmt.Sprintf("🍵 AgentTea v%s - Ollama Cloud TUI", api.Version))
	titleBar = lipgloss.NewStyle().Width(m.width).Render(titleBar)

	if m.showHelp {
		helpView := m.renderHelp()
		statusBar := ui.StatusBarStyle.Render("Esc: 关闭帮助")
		return lipgloss.JoinVertical(lipgloss.Left,
			titleBar,
			helpView,
			statusBar,
		)
	}

	chatContent := m.renderMessages()
	if m.loading {
		chatContent += "\n" + m.spinner.View() + " 正在生成响应..."
	}
	m.viewport.SetContent(chatContent)

	inputView := m.textarea.View()
	if m.focused == FocusInput {
		inputView = ui.InputBorderFocusedStyle.Render(inputView)
	} else {
		inputView = ui.InputBorderStyle.Render(inputView)
	}

	roundCount := 0
	for _, msg := range m.messages {
		if msg.Role == "user" {
			roundCount++
		}
	}

	modelInfo := ui.ModelInfoStyle.Render(fmt.Sprintf("模型: %s | 轮次: %d", m.client.Model, roundCount))
	shortcutInfo := "Ctrl+Enter: 发送 | Tab: 切换焦点 | Ctrl+M: 切换模型 | Ctrl+C: 退出"
	if m.loading {
		shortcutInfo = "Esc: 取消请求 | Ctrl+C: 退出"
	}
	if m.hasError {
		shortcutInfo = "r: 重试 | Ctrl+C: 退出"
	}
	statusBar := lipgloss.JoinHorizontal(lipgloss.Top, modelInfo, "  ", ui.StatusBarStyle.Render(shortcutInfo))

	availableHeight := m.height - lipgloss.Height(titleBar) - lipgloss.Height(inputView) - lipgloss.Height(statusBar)
	if availableHeight < 5 {
		availableHeight = 5
	}
	m.viewport.Height = availableHeight
	chatView := m.viewport.View()

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		chatView,
		inputView,
		statusBar,
	)

	if m.showModelPicker {
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(m.width).Height(m.height).Render(mainView),
		) + "\n" + m.renderModelPicker()
	}

	if m.showConvPicker {
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(m.width).Height(m.height).Render(mainView),
		) + "\n" + m.renderConvPicker()
	}

	if m.showPromptPicker {
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(m.width).Height(m.height).Render(mainView),
		) + "\n" + m.renderPromptPicker()
	}

	if m.showNotePicker {
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(m.width).Height(m.height).Render(mainView),
		) + "\n" + m.renderNotePicker()
	}

	if m.showNoteEditor {
		if m.noteEditorMode == "view" {
			return lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Width(m.width).Height(m.height).Render(mainView),
			) + "\n" + m.renderNoteViewer()
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Width(m.width).Height(m.height).Render(mainView),
		) + "\n" + m.renderNoteEditor()
	}

	return mainView
}
