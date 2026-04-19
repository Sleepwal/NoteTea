# AgentTea

基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 框架的终端 AI 聊天 TUI 应用，支持 Ollama Cloud 和 OpenAI 兼容 API。

通过直观的终端界面与大语言模型交互，支持流式响应、Markdown 渲染、多轮对话、主题切换和插件扩展。

## 功能特性

- **流式响应** — 实时逐 token 显示模型输出，增量渲染优化
- **Markdown 渲染** — 代码高亮、表格、列表等完整 Markdown 支持
- **多轮对话** — 自动维护对话上下文，支持对话持久化与恢复
- **多 API 后端** — Ollama Cloud (NDJSON) 和 OpenAI 兼容 API (SSE) 双模式
- **主题系统** — dark / light / catppuccin 三套主题，Ctrl+T 即时切换
- **对话管理** — 新建、切换、导出、删除对话
- **System Prompt** — 内置 4 套预设，支持自定义
- **代码块复制** — Ctrl+Y 一键复制最后一个代码块到剪贴板
- **插件机制** — before_send / after_receive / on_error 钩子扩展
- **输入历史** — ↑/↓ 浏览历史输入
- **Token 统计** — 流式响应结束显示 prompt/completion token 数和耗时
- **错误恢复** — 友好的错误提示和 r 键重试机制

## 快速开始

### 前置条件

- Go 1.22+
- Ollama Cloud 或 OpenAI 兼容 API 密钥

### 安装

```bash
git clone https://github.com/user/agenttea.git
cd agenttea
go build -o agenttea .
```

### 配置

#### 方式一：环境变量（推荐）

```bash
# Linux / macOS
export OLLAMA_API_KEY="your_api_key_here"

# Windows CMD
set OLLAMA_API_KEY=your_api_key_here

# Windows PowerShell
$env:OLLAMA_API_KEY = "your_api_key_here"
```

在 [ollama.com/settings/keys](https://ollama.com/settings/keys) 获取 Ollama Cloud API 密钥。

#### 方式二：配置文件

配置文件路径：`~/.agenttea/config.json`

首次运行会自动创建默认配置。手动创建示例：

```json
{
  "base_url": "https://ollama.com",
  "model": "qwen3.5:cloud",
  "temperature": 0.7,
  "num_predict": 4096,
  "theme": "dark",
  "system_prompt": "",
  "prompt_presets": [
    { "name": "默认助手", "prompt": "你是一个有帮助的AI助手。" },
    { "name": "代码助手", "prompt": "你是一个专业的编程助手..." }
  ],
  "hooks": [
    { "type": "after_receive", "command": "notify-send 'AgentTea' '响应完成'", "enabled": true }
  ]
}
```

> **安全提示**：`api_key` 字段不会持久化到配置文件中，API 密钥仅通过环境变量或运行时内存持有。

#### 配置项说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `base_url` | string | `https://ollama.com` | API 基础 URL，自动检测后端类型 |
| `model` | string | `qwen3.5:cloud` | 默认模型名称 |
| `temperature` | float64 | `0.7` | 生成温度 (0.0-1.0) |
| `num_predict` | int | `4096` | 最大生成 token 数 |
| `theme` | string | `dark` | 主题: `dark` / `light` / `catppuccin` |
| `system_prompt` | string | `""` | 全局 System Prompt |
| `prompt_presets` | array | 4 个内置预设 | System Prompt 预设列表 |
| `hooks` | array | `[]` | 插件钩子配置 |

#### 使用 OpenAI 兼容 API

将 `base_url` 设置为 OpenAI 兼容的端点即可，AgentTea 会自动检测并切换为 SSE 流式格式：

```json
{
  "base_url": "https://api.openai.com",
  "model": "gpt-4o"
}
```

```bash
export OPENAI_API_KEY="your_openai_key"
```

> 后端检测规则：URL 包含 `ollama.com` 使用 Ollama NDJSON 格式，否则使用 OpenAI SSE 格式。

### 运行

```bash
go run .
# 或指定模型
go run . --model deepseek:v3.1:571b

# 编译后运行
./agenttea
./agenttea --model gpt-oss:120b
```

### 查看版本

```bash
./agenttea --version
```

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+Enter` / `Alt+Enter` | 发送消息 |
| `Enter` | 输入区内换行 |
| `Tab` | 切换输入区/对话区焦点 |
| `↑` / `↓` (输入区) | 浏览输入历史 |
| `↑` / `↓` (对话区) | 滚动对话 |
| `Ctrl+L` | 清空对话历史 |
| `Ctrl+N` | 新建对话 |
| `Ctrl+P` | 打开对话列表 |
| `Ctrl+E` | 导出当前对话为 Markdown |
| `Ctrl+S` | 切换 System Prompt 预设 |
| `Ctrl+Y` | 复制最后一个代码块到剪贴板 |
| `Ctrl+T` | 切换主题 (dark/light/catppuccin) |
| `Ctrl+M` | 切换模型 |
| `r` | 重试上次请求（出错时） |
| `Esc` | 取消当前请求 / 关闭弹窗 |
| `?` | 显示帮助（对话区焦点时） |
| `Ctrl+C` | 退出应用 |

> **Windows 用户注意**：Windows 上 `Ctrl+Enter` 可能无法识别，请使用 `Alt+Enter` 发送消息。需在 Windows Terminal 设置中禁用 Alt+Enter 的全屏切换。

## 项目架构

### 目录结构

```
AgentTea/
├── main.go                        # 入口：flag 解析、初始化、启动 Bubble Tea
├── internal/
│   ├── api/
│   │   ├── client.go              # API 客户端：双后端 (Ollama/OpenAI) 流式请求
│   │   ├── client_test.go         # 客户端单元测试
│   │   └── version.go             # 版本号常量
│   ├── config/
│   │   ├── config.go              # 配置加载/保存/默认值
│   │   └── config_test.go         # 配置单元测试
│   ├── logger/
│   │   └── logger.go              # 文件日志：Info/Error/Debug
│   ├── model/
│   │   ├── app.go                 # Bubble Tea 主模型：Update/View、消息分发
│   │   ├── app_test.go            # 模型单元测试
│   │   ├── chat.go                # 聊天逻辑：发送/流式读取/重试/持久化/导出
│   │   ├── handlers.go            # 键盘事件处理：所有快捷键和弹窗交互
│   │   ├── layout.go              # 布局计算：viewport/textarea 尺寸
│   │   └── render.go              # 渲染逻辑：消息列表/帮助/弹窗
│   ├── msg/
│   │   └── messages.go            # Bubble Tea 自定义消息类型
│   ├── plugin/
│   │   └── plugin.go              # 插件系统：Hook 管理器和 Shell 命令执行
│   ├── store/
│   │   └── store.go               # 对话持久化：JSON 文件存储
│   └── ui/
│       ├── clipboard.go           # 剪贴板操作：代码块提取和复制
│       ├── keymap.go              # 快捷键绑定定义
│       ├── markdown.go            # Markdown 渲染：Glamour 集成
│       └── styles.go              # 主题系统：动态样式和三套预设主题
```

### 架构图

```
┌─────────────────────────────────────────────────────────┐
│                       main.go                           │
│              flag 解析 → 初始化 → tea.NewProgram         │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    model/app.go                          │
│              Bubble Tea Model (Update/View)              │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │handlers  │  │  chat    │  │  render  │              │
│  │键盘事件   │  │聊天逻辑   │  │渲染逻辑   │              │
│  └──────────┘  └────┬─────┘  └──────────┘              │
│                      │                                   │
└──────────────────────┼──────────────────────────────────┘
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
   ┌────────────┐ ┌──────────┐ ┌──────────┐
   │  api/      │ │ store/   │ │ plugin/  │
   │ API 客户端  │ │ 对话存储  │ │ 插件钩子  │
   │ 双后端支持  │ │ JSON 文件│ │ Shell 执行│
   └─────┬──────┘ └──────────┘ └──────────┘
         │
         ▼
   ┌──────────┐     ┌──────────┐
   │ Ollama   │     │ OpenAI   │
   │ NDJSON   │     │ SSE      │
   │ /api/chat│     │ /v1/chat │
   └──────────┘     └──────────┘

   ┌──────────┐ ┌──────────┐ ┌──────────┐
   │ config/  │ │ logger/  │ │   ui/    │
   │ 配置管理  │ │ 文件日志  │ │ 主题/MD  │
   └──────────┘ └──────────┘ │ 剪贴板    │
                              └──────────┘
```

### 数据流

```
用户输入 → handleSend() → buildAPIMessages() → client.SendChat()
                                                      │
                                                      ▼
                                              HTTP 流式响应
                                                      │
                        ┌─────────────────────────────┤
                        ▼                             ▼
                 StreamTokenMsg                StreamDoneMsg
                        │                             │
                        ▼                             ▼
              增量渲染到 viewport          设置 Stats → 保存对话
              (renderedPrefix 缓存)       → 执行 after_receive 钩子
```

## 可用模型

默认使用 `qwen3.5:cloud` 模型，运行时可通过 `Ctrl+M` 切换。

Ollama Cloud 可用模型：
- `qwen3.5:cloud`
- `gpt-oss:120b`
- `gpt-oss:20b`
- `deepseek:v3.1:571b`

使用 OpenAI 兼容 API 时可使用对应平台支持的任意模型。

## 插件系统

通过 `hooks` 配置在特定事件触发时执行 Shell 命令：

| 钩子类型 | 触发时机 | 输入数据 |
|----------|----------|----------|
| `before_send` | 发送消息前 | 用户输入文本 |
| `after_receive` | 收到完整响应后 | 助手回复内容 |
| `on_error` | 请求出错时 | 错误信息 |

配置示例：

```json
{
  "hooks": [
    { "type": "after_receive", "command": "notify-send 'AgentTea' '响应完成'", "enabled": true },
    { "type": "on_error", "command": "echo '%s' >> ~/agenttea_errors.log", "enabled": true }
  ]
}
```

> 钩子命令通过 `sh -c` 执行，数据通过 stdin 管道传入，异步运行不阻塞主界面。

## 数据存储

所有数据存储在 `~/.agenttea/` 目录下：

```
~/.agenttea/
├── config.json              # 配置文件 (权限 0600，不含 API 密钥)
├── conversations/           # 对话记录 (JSON，权限 0600)
│   ├── 1745000000000.json
│   └── ...
├── exports/                 # 导出的 Markdown 文件
│   └── ...
└── logs/                    # 运行日志
    └── agenttea_2026-04-19.log
```

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.22+ |
| TUI 框架 | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | v1.3.10 |
| UI 组件 | [Bubbles](https://github.com/charmbracelet/bubbles) | v1.0.0 |
| 样式 | [Lip Gloss](https://github.com/charmbracelet/lipgloss) | v1.1.0 |
| Markdown | [Glamour](https://github.com/charmbracelet/glamour) | v1.0.0 |
| 剪贴板 | [atotto/clipboard](https://github.com/atotto/clipboard) | v0.1.4 |

## 贡献指南

欢迎贡献！请遵循以下流程：

1. **Fork** 本仓库
2. 创建功能分支：`git checkout -b feat/your-feature`
3. 提交代码，遵循提交规范：
   ```
   <类型>(<作用域>): <中文描述>
   ```
   类型前缀：`feat` / `fix` / `docs` / `style` / `refactor` / `test` / `chore`
4. 确保通过所有测试：`go test ./...`
5. 确保构建成功：`go build ./...`
6. 推送分支并创建 **Pull Request**

### 开发环境搭建

```bash
git clone https://github.com/user/agenttea.git
cd agenttea
go mod download
go build -o agenttea .
```

### 代码规范

- 遵循 Go 标准格式化：`gofmt` / `goimports`
- 单一职责提交：每个 PR 只包含一个功能或修复
- 新功能需附带测试
- 提交信息使用中文描述

## 许可证

MIT License
