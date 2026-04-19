## #003 [已解决] 重新载入聊天结果后界面不能滚动

| 字段 | 内容 |
|------|------|
| **类型** | 运行时错误（逻辑缺陷） |
| **发生时间** | 2026-04-19 |
| **涉及文件** | `internal/model/app.go:158-166`、`internal/model/app.go:78-138` |
| **错误信息** | 重新载入上一次聊天结果后，聊天结果界面不能滚动 |
| **堆栈跟踪** | 不适用（非 panic，为 viewport 内容未同步） |
| **相关上下文** | `NewAppModel()` 恢复了消息但从未调用 `viewport.SetContent()`；`View()` 方法使用值接收者，其中 `SetContent()` 修改的是副本；`WindowSizeMsg` 处理器只调用了 `recalcLayout()` 没有同步 viewport 内容 |

### 根因分析

Bubble Tea 的 `Update()` 方法使用值接收者返回新模型。`View()` 方法也是值接收者，其中调用 `m.viewport.SetContent()` 修改的是副本，实际模型中的 viewport 没有内容。当滚动键事件通过 `delegateToComponents()` 传给 viewport 时，viewport 内部没有内容所以无法滚动。

具体问题点：
1. `NewAppModel()` 恢复了 `messages` 但从未调用 `viewport.SetContent()` 设置内容
2. `WindowSizeMsg` 处理器只调用了 `recalcLayout()` 调整尺寸，没有同步 viewport 内容
3. `View()` 中每帧设置 viewport 内容，但这只影响渲染副本，不影响模型中的 viewport 状态

### 解决方案

在 `WindowSizeMsg` 处理器中，`recalcLayout()` 之后立即调用 `m.viewport.SetContent(m.renderMessages())` 和 `m.viewport.GotoBottom()`，确保模型中的 viewport 在窗口尺寸确定后就有正确的内容。bubbles viewport 的 `SetContent()` 不会重置 YOffset（仅在超出范围时调整），所以 `View()` 中每帧调用也是安全的。

## #002 [已解决] Token 统计信息中混入 ANSI 转义序列

| 字段 | 内容 |
|------|------|
| **类型** | 运行时错误（渲染缺陷） |
| **发生时间** | 2026-04-19 |
| **涉及文件** | `internal/model/chat.go:295`、`internal/model/render.go:39-44` |
| **错误信息** | 流式响应结束时显示的 token 统计信息中混入了 ANSI 转义序列（如 `[3;38;2;136;136;136m`），显示为乱码文本 |
| **堆栈跟踪** | 不适用（非 panic，为样式渲染逻辑错误） |
| **相关上下文** | `handleStreamDone()` 中使用 `ui.StatsStyle.Render()` 将带 ANSI 样式的文本直接追加到 `ChatMessage.Content` 字段；之后 `renderMessages()` 对已完成的 assistant 消息调用 `ui.RenderMarkdown()` (glamour) 处理时，glamour 无法理解嵌入的 ANSI 码 |

### 根因分析

`StatsStyle.Render()` 将 ANSI 转义码（颜色/斜体样式）直接嵌入到 `Content` 字符串字段中。之后在 `renderMessages()` 中，已完成的 assistant 消息会经过 `ui.RenderMarkdown()` (glamour) 处理，glamour 将 ANSI 转义码当作普通文本字符处理，导致终端显示 `[3;38;2;136;136;136m` 这样的乱码。

核心问题是**数据与样式耦合**：`Content` 字段应存储纯文本数据，样式应在渲染层应用。

### 解决方案

将 Stats 从 Content 中分离为独立字段：
1. `ChatMessage` 新增 `Stats string` 字段，存储纯文本统计信息
2. `handleStreamDone()` 中将原始统计文本存入 `last.Stats`，不再追加带样式的文本到 `Content`
3. `renderMessages()` 和 `renderMessagesExceptLast()` 中，在 Markdown 渲染之后单独用 `StatsStyle.Render()` 渲染 Stats
4. `StoreMessage` 新增 `Stats` 字段（`json:"stats,omitempty"`），支持持久化
5. 所有消息恢复点同步更新 Stats 字段

## #001 [已解决] Windows 上 Ctrl+Enter 无法发送消息

| 字段 | 内容 |
|------|------|
| **类型** | 运行时错误（平台兼容性问题） |
| **发生时间** | 2026-04-18 |
| **涉及文件** | `internal/model/app.go:163`、`internal/ui/keymap.go:22-24` |
| **错误信息** | 在 Windows 系统上按下 Ctrl+Enter 无法触发发送消息功能，按键被 textarea 组件当作普通 Enter 处理（插入换行） |
| **堆栈跟踪** | 不适用（非 panic，为按键事件未正确匹配） |
| **相关上下文** | 应用使用 Bubble Tea v1.3.10 框架，在 Windows 上通过 coninput 读取键盘事件 |

### 根因分析

Bubble Tea 库在 Windows 平台的按键处理存在缺陷。在 `key_windows.go` 的 `keyType()` 函数中，`VK_RETURN` 的处理没有检查 Ctrl 修饰键：

```go
case coninput.VK_RETURN:
    return KeyEnter  // ❌ 未区分 Ctrl+Enter 和 Enter
```

对比箭头键的正确处理方式（检查了 Ctrl 修饰键）：

```go
case coninput.VK_UP:
    switch {
    case ctrlPressed: return KeyCtrlUp  // ✅ 正确检查了 Ctrl
    default:          return KeyUp
    }
```

此外，`KeyMsg` 结构体只有 `Alt` 字段，没有 `Ctrl` 字段，因此应用层也无法通过其他方式区分 Ctrl+Enter 和普通 Enter。这导致 `case "ctrl+enter":` 在 Windows 上永远不会匹配。

### 解决方案

添加 `alt+enter` 作为替代发送快捷键。在 Windows 上，`Alt` 修饰键被正确设置到 `KeyMsg.Alt` 字段中，`alt+enter` 能被正确检测为 `"alt+enter"` 字符串。

修改内容：
1. `internal/model/app.go` — `handleKeyMsg` 中将 `case "ctrl+enter"` 改为 `case "ctrl+enter", "alt+enter"`
2. `internal/model/app.go` — 更新状态栏、欢迎消息、帮助页面中的快捷键提示文字
3. `internal/ui/keymap.go` — `Send` 绑定添加 `"alt+enter"` 键

> 注意：Windows Terminal 中 Alt+Enter 默认用于全屏切换，需在终端设置中禁用或重新映射该快捷键。
