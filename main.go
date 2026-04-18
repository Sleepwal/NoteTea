package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/model"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("AgentTea v%s\n", api.Version)
		os.Exit(0)
	}

	client, err := api.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		fmt.Fprintln(os.Stderr, "请设置 OLLAMA_API_KEY 环境变量后重试。")
		fmt.Fprintln(os.Stderr, "  export OLLAMA_API_KEY=your_api_key  (Linux/macOS)")
		fmt.Fprintln(os.Stderr, "  set OLLAMA_API_KEY=your_api_key     (Windows CMD)")
		fmt.Fprintln(os.Stderr, "  $env:OLLAMA_API_KEY=\"your_api_key\"  (Windows PowerShell)")
		os.Exit(1)
	}

	appModel := model.NewAppModel(client)

	p := tea.NewProgram(
		appModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}
}
