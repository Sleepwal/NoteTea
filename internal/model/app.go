package model

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/msg"
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

	focused  FocusArea
	loading  bool
	showHelp bool

	messages     []ChatMessage
	inputHistory []string
	historyIndex int

	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	client       *api.Client
	streamReader io.ReadCloser
	streamScan   *bufio.Scanner
	cancelFunc   context.CancelFunc

	apiMessages []api.Message
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

	return AppModel{
		focused:      FocusInput,
		loading:      false,
		showHelp:     false,
		messages:     []ChatMessage{},
		inputHistory: []string{},
		historyIndex: -1,
		textarea:     ta,
		viewport:     vp,
		spinner:      sp,
		client:       client,
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
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			last.Streaming = false
		}
		m.loading = false
		m.cancelFunc = nil
		m.cleanupStream()
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case msg.ApiErrorMsg:
		m.loading = false
		m.cancelFunc = nil
		m.cleanupStream()
		if len(m.messages) > 0 {
			last := &m.messages[len(m.messages)-1]
			if last.Streaming {
				last.Streaming = false
				last.Content += fmt.Sprintf("\n\n[错误] %s", message.Err.Error())
			}
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	return m.delegateToComponents(teaMsg)
}

func (m AppModel) handleKeyMsg(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "ctrl+c":
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

	case "esc":
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

	statusText := "Ctrl+Enter/Alt+Enter: 发送 | Tab: 切换焦点 | Ctrl+C: 退出"
	if m.loading {
		statusText = "Esc: 取消请求 | Ctrl+C: 退出"
	}
	statusBar := ui.StatusBarStyle.Render(statusText)

	availableHeight := m.height - lipgloss.Height(titleBar) - lipgloss.Height(inputView) - lipgloss.Height(statusBar)
	if availableHeight < 5 {
		availableHeight = 5
	}
	m.viewport.Height = availableHeight
	chatView := m.viewport.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		chatView,
		inputView,
		statusBar,
	)
}

func (m *AppModel) recalcLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	m.viewport.Width = m.width - 4
	m.textarea.SetWidth(m.width - 6)

	titleHeight := 1
	inputHeight := 7
	statusHeight := 1
	chatHeight := m.height - titleHeight - inputHeight - statusHeight
	if chatHeight < 5 {
		chatHeight = 5
	}
	m.viewport.Height = chatHeight
}

func (m AppModel) handleSend() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	m.inputHistory = append(m.inputHistory, input)
	m.historyIndex = -1

	userMsg := ChatMessage{
		Role:      "user",
		Content:   input,
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

	m.textarea.Reset()
	m.loading = true
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()

	return m, m.startChatRequest()
}

func (m AppModel) buildAPIMessages() []api.Message {
	apiMsgs := make([]api.Message, 0, len(m.messages))
	for _, chatMsg := range m.messages {
		if chatMsg.Role == "user" || (chatMsg.Role == "assistant" && !chatMsg.Streaming) {
			apiMsgs = append(apiMsgs, api.Message{
				Role:    chatMsg.Role,
				Content: chatMsg.Content,
			})
		}
	}
	return apiMsgs
}

func (m AppModel) startChatRequest() tea.Cmd {
	apiMsgs := m.apiMessages
	client := m.client

	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())

		reader, err := client.SendChat(ctx, apiMsgs)
		if err != nil {
			cancel()
			return msg.ApiErrorMsg{Err: err}
		}

		return msg.StreamStartMsg{
			Reader:    reader,
			CancelCtx: cancel,
		}
	}
}

func (m AppModel) readNextStreamToken() tea.Cmd {
	scanner := m.streamScan

	return func() tea.Msg {
		if scanner == nil {
			return msg.StreamDoneMsg{}
		}

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var resp api.ChatResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}

			if resp.Done {
				return msg.StreamDoneMsg{
					TotalDuration:   resp.TotalDuration,
					PromptEvalCount: resp.PromptEvalCount,
					EvalCount:       resp.EvalCount,
				}
			}

			if resp.Message.Content != "" {
				return msg.StreamTokenMsg{Content: resp.Message.Content}
			}
		}

		if err := scanner.Err(); err != nil {
			return msg.ApiErrorMsg{Err: fmt.Errorf("读取流式响应失败: %w", err)}
		}

		return msg.StreamDoneMsg{}
	}
}

func (m *AppModel) cleanupStream() {
	if m.streamReader != nil {
		m.streamReader.Close()
		m.streamReader = nil
	}
	m.streamScan = nil
}

func (m AppModel) renderMessages() string {
	var sb strings.Builder

	if len(m.messages) == 0 {
		sb.WriteString(ui.HelpStyle.Render("欢迎使用 AgentTea！输入消息开始对话。"))
		sb.WriteString("\n")
		sb.WriteString(ui.HelpStyle.Render("按 Ctrl+Enter 或 Alt+Enter 发送消息，Tab 切换焦点，? 查看帮助。"))
		return sb.String()
	}

	for i, chatMsg := range m.messages {
		if i > 0 {
			sb.WriteString("\n")
		}

		switch chatMsg.Role {
		case "user":
			prefix := ui.UserPrefixStyle.Render("[You]")
			content := ui.UserMsgStyle.Render(chatMsg.Content)
			sb.WriteString(fmt.Sprintf("%s %s\n", prefix, content))
		case "assistant":
			prefix := ui.AssistantPrefixStyle.Render("[Assistant]")
			if chatMsg.Streaming && chatMsg.Content == "" {
				sb.WriteString(fmt.Sprintf("%s ...\n", prefix))
			} else {
				content := ui.AssistantMsgStyle.Render(chatMsg.Content)
				sb.WriteString(fmt.Sprintf("%s %s\n", prefix, content))
			}
		case "system":
			content := ui.SystemMsgStyle.Render(chatMsg.Content)
			sb.WriteString(content + "\n")
		}
	}

	return sb.String()
}

func (m AppModel) renderHelp() string {
	helpText := `
快捷键说明:
  Ctrl+Enter/Alt+Enter    发送消息
  Tab           切换输入区/对话区焦点
  ↑ / ↓         在对话区滚动
  Ctrl+L        清空对话历史
  Ctrl+N        新建对话
  Esc           取消当前请求 / 关闭帮助
  ?             显示/隐藏帮助（对话区焦点时）
  Ctrl+C        退出应用

当前模型: ` + m.client.Model + `
`
	return ui.HelpStyle.Render(helpText)
}
