package plugin

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/user/agenttea/internal/logger"
)

type HookType string

const (
	HookBeforeSend HookType = "before_send"
	HookAfterReceive HookType = "after_receive"
	HookOnError     HookType = "on_error"
)

type Hook struct {
	Type     HookType
	Command  string
	Enabled  bool
}

type Manager struct {
	hooks []Hook
}

func NewManager() *Manager {
	return &Manager{
		hooks: []Hook{},
	}
}

func (m *Manager) AddHook(hookType HookType, command string) {
	m.hooks = append(m.hooks, Hook{
		Type:    hookType,
		Command: command,
		Enabled: true,
	})
}

func (m *Manager) ExecuteHooks(hookType HookType, data string) {
	for i, hook := range m.hooks {
		if hook.Type == hookType && hook.Enabled {
			go m.executeCommand(i, hook, data)
		}
	}
}

func (m *Manager) executeCommand(index int, hook Hook, data string) {
	cmd := exec.Command("sh", "-c", hook.Command)
	cmd.Stdin = strings.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Hook %d (%s) 执行失败: %v, 输出: %s", index, hook.Type, err, string(output))
		return
	}
	logger.Info("Hook %d (%s) 执行成功, 输出: %s", index, hook.Type, strings.TrimSpace(string(output)))
}

func (m *Manager) ListHooks() []Hook {
	return m.hooks
}

func (m *Manager) EnableHook(index int) error {
	if index < 0 || index >= len(m.hooks) {
		return fmt.Errorf("hook 索引 %d 超出范围", index)
	}
	m.hooks[index].Enabled = true
	return nil
}

func (m *Manager) DisableHook(index int) error {
	if index < 0 || index >= len(m.hooks) {
		return fmt.Errorf("hook 索引 %d 超出范围", index)
	}
	m.hooks[index].Enabled = false
	return nil
}
