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
		})
	}

	cfg, _ := config.Load()
	var sysPrompt string
	var presets []config.SystemPromptPreset
	if cfg != nil {
		sysPrompt = cfg.SystemPrompt
		presets = cfg.PromptPresets
	}
	if len(presets) == 0 {
		presets = config.DefaultConfig.PromptPresets
	}

	return AppModel{
		focused:       FocusInput,
		loading:       false,
		showHelp:      false,
		messages:      restoreMsgs,
		inputHistory:  []string{},
		historyIndex:  -1,
		textarea:      ta,
		viewport:      vp,
		spinner:       sp,
		client:        client,
		conversation:  conv,
		systemPrompt:  sysPrompt,
		promptPresets: presets,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m AppModel) Update(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := teaMsg.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.recalcLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(message)

	case msg.StreamStartMsg:
		m.streamReader = message.Reader
		m.cancelFunc = message.CancelCtx
		m.streamScan = bufio.NewScanner(m.streamReader)
		m.streamScan.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		return m, m.readNextStreamToken()

	case msg.StreamTokenMsg:
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			if last.Streaming {
				last.Content += message.Content
				m.viewport.SetContent(m.renderMessages())
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

	return m.delegateToComponents(teaMsg)
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

	return mainView
}
