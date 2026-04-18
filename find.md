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
