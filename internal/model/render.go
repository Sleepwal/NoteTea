package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/ui"
)

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
			} else if chatMsg.Streaming {
				content := ui.AssistantMsgStyle.Render(chatMsg.Content)
				sb.WriteString(fmt.Sprintf("%s %s\n", prefix, content))
			} else {
				content := ui.RenderMarkdown(chatMsg.Content)
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
  ↑ / ↓ (输入区)  浏览输入历史
  ↑ / ↓ (对话区)  滚动对话
  Tab           切换输入区/对话区焦点
  Ctrl+L        清空对话历史
  Ctrl+N        新建对话
  Ctrl+P        打开对话列表
  Ctrl+E        导出当前对话
  Ctrl+S        切换 System Prompt 预设
  Ctrl+M        切换模型
  Esc           取消当前请求 / 关闭帮助
  ?             显示/隐藏帮助（对话区焦点时）
  Ctrl+C        退出应用

当前模型: ` + m.client.Model + `
`
	return ui.HelpStyle.Render(helpText)
}

func (m AppModel) renderModelPicker() string {
	var sb strings.Builder
	sb.WriteString("选择模型:\n\n")

	for i, model := range api.AvailableModels {
		cursor := "  "
		style := ui.HelpStyle
		if i == m.modelCursor {
			cursor = "> "
			style = ui.ModelPickerActiveStyle
		}
		label := model
		if model == m.client.Model {
			label += " (当前)"
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}

	sb.WriteString("\n")
	sb.WriteString(ui.HelpStyle.Render("↑/k ↑/j 导航 | Enter 确认 | Esc 取消"))

	content := sb.String()
	dialog := ui.ModelPickerBorderStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}

func (m AppModel) renderPromptPicker() string {
	var sb strings.Builder
	sb.WriteString("选择 System Prompt 预设:\n\n")

	for i, preset := range m.promptPresets {
		cursor := "  "
		style := ui.HelpStyle
		if i == m.promptCursor {
			cursor = "> "
			style = ui.ModelPickerActiveStyle
		}

		isActive := m.systemPrompt == preset.Prompt
		label := fmt.Sprintf("%s: %s", preset.Name, preset.Prompt)
		if len(label) > 60 {
			label = label[:60] + "..."
		}
		if isActive {
			label += " (当前)"
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}

	sb.WriteString("\n")
	sb.WriteString(ui.HelpStyle.Render("↑/k ↓/j 导航 | Enter 选择 | c 清除 | Esc 取消"))

	content := sb.String()
	dialog := ui.ModelPickerBorderStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}
func (m AppModel) renderConvPicker() string {
	var sb strings.Builder
	sb.WriteString("对话列表:\n\n")

	for i, conv := range m.convList {
		cursor := "  "
		style := ui.HelpStyle
		if i == m.convCursor {
			cursor = "> "
			style = ui.ModelPickerActiveStyle
		}

		isCurrent := m.conversation != nil && conv.ID == m.conversation.ID
		label := fmt.Sprintf("%s  (%s, %d条消息)", conv.Title, conv.UpdatedAt.Format("01-02 15:04"), len(conv.Messages))
		if isCurrent {
			label += " (当前)"
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}

	sb.WriteString("\n")
	sb.WriteString(ui.HelpStyle.Render("↑/k ↓/j 导航 | Enter 切换 | d 删除 | Esc 取消"))

	content := sb.String()
	dialog := ui.ModelPickerBorderStyle.Render(content)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}
