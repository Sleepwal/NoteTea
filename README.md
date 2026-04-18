# AgentTea

基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 框架的 Ollama Cloud 终端用户界面（TUI）应用。

通过直观的终端界面与大语言模型进行交互，支持流式响应、多轮对话和多行输入。

## 功能特性

- **流式响应** — 实时逐 token 显示模型输出
- **多轮对话** — 自动维护对话上下文
- **多行输入** — Enter 换行，Ctrl+Enter 发送
- **请求取消** — Esc 键随时中断生成
- **对话管理** — 清空历史、新建对话
- **错误恢复** — 友好的错误提示和重试机制
- **优雅退出** — 资源自动清理

## 快速开始

### 前置条件

- Go 1.22+
- [Ollama Cloud](https://ollama.com) API 密钥

### 安装

```bash
git clone https://github.com/user/agenttea.git
cd agenttea
go build -o agenttea .
```

### 配置

设置 API 密钥环境变量：

```bash
# Linux / macOS
export OLLAMA_API_KEY="your_api_key_here"

# Windows CMD
set OLLAMA_API_KEY=your_api_key_here

# Windows PowerShell
$env:OLLAMA_API_KEY = "your_api_key_here"
```

在 [ollama.com/settings/keys](https://ollama.com/settings/keys) 获取 API 密钥。

### 运行

```bash
go run .
# 或
./agenttea
```

### 查看版本

```bash
./agenttea --version
```

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+Enter` | 发送消息 |
| `Enter` | 输入区内换行 |
| `Tab` | 切换输入区/对话区焦点 |
| `↑` / `↓` | 对话区滚动 |
| `Ctrl+L` | 清空对话历史 |
| `Ctrl+N` | 新建对话 |
| `Esc` | 取消当前请求 / 关闭帮助 |
| `?` | 显示帮助（对话区焦点时） |
| `Ctrl+C` | 退出应用 |

## 项目结构

```
AgentTea/
├── main.go                     # 入口文件
├── internal/
│   ├── api/
│   │   ├── client.go           # Ollama API 客户端
│   │   └── version.go          # 版本号定义
│   ├── model/
│   │   └── app.go              # Bubble Tea 主模型
│   ├── ui/
│   │   ├── styles.go           # 样式定义
│   │   └── keymap.go           # 快捷键绑定
│   └── msg/
│       └── messages.go         # 自定义消息类型
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go |
| TUI 框架 | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| UI 组件 | [Bubbles](https://github.com/charmbracelet/bubbles) |
| 样式 | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| API | Ollama Cloud Chat API |

## 默认模型

默认使用 `qwen3.5:cloud` 模型。可在 `internal/api/client.go` 中修改 `DefaultModel` 常量切换模型。

可用模型：
- `qwen3.5:cloud`
- `gpt-oss:120b`
- `gpt-oss:20b`
- `deepseek:v3.1:571b`

## 许可证

MIT License
