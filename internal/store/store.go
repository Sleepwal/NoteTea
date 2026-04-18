package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type StoreMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Conversation struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Model     string        `json:"model"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Messages  []StoreMessage `json:"messages"`
}

func storeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".agenttea", "conversations"), nil
}

func ensureDir() error {
	dir, err := storeDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

func SaveConversation(conv *Conversation) error {
	if err := ensureDir(); err != nil {
		return err
	}

	conv.UpdatedAt = time.Now()

	path := filepath.Join(storeDirMust(), conv.ID+".json")
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化对话失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("写入对话文件失败: %w", err)
	}

	return nil
}

func LoadConversation(id string) (*Conversation, error) {
	dir, err := storeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取对话文件失败: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("解析对话文件失败: %w", err)
	}

	return &conv, nil
}

func ListConversations() ([]Conversation, error) {
	dir, err := storeDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取对话目录失败: %w", err)
	}

	var convs []Conversation
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		conv, err := LoadConversation(id)
		if err != nil {
			continue
		}
		if conv != nil {
			convs = append(convs, *conv)
		}
	}

	return convs, nil
}

func DeleteConversation(id string) error {
	dir, err := storeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, id+".json")
	return os.Remove(path)
}

func NewConversation(modelName string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        fmt.Sprintf("%d", now.UnixMilli()),
		Title:     "新对话",
		Model:     modelName,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []StoreMessage{},
	}
}

func LoadLastConversation() (*Conversation, error) {
	convs, err := ListConversations()
	if err != nil {
		return nil, err
	}
	if len(convs) == 0 {
		return nil, nil
	}

	latest := convs[0]
	for _, c := range convs[1:] {
		if c.UpdatedAt.After(latest.UpdatedAt) {
			latest = c
		}
	}

	return LoadConversation(latest.ID)
}

func storeDirMust() string {
	dir, _ := storeDir()
	return dir
}
