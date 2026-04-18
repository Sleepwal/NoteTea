package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/config"
	"github.com/user/agenttea/internal/model"
)

func main() {
	showVersion := flag.Bool("version", false, "显示版本号")
	modelName := flag.String("model", "", "指定模型名称")
	flag.Parse()

	if *showVersion {
		fmt.Printf("AgentTea v%s\n", api.Version)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	if *modelName != "" {
		cfg.Model = *modelName
	}

	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "错误: API 密钥未设置")
		fmt.Fprintln(os.Stderr, "请通过以下方式之一设置 API 密钥:")
		fmt.Fprintln(os.Stderr, "  1. 配置文件 ~/.agenttea/config.json 中的 api_key 字段")
		fmt.Fprintln(os.Stderr, "  2. 环境变量 OLLAMA_API_KEY")
		fmt.Fprintln(os.Stderr, "    export OLLAMA_API_KEY=your_api_key  (Linux/macOS)")
		fmt.Fprintln(os.Stderr, "    set OLLAMA_API_KEY=your_api_key     (Windows CMD)")
		fmt.Fprintln(os.Stderr, "    $env:OLLAMA_API_KEY=\"your_api_key\"  (Windows PowerShell)")
		os.Exit(1)
	}

	client := api.NewClientWithConfig(cfg.BaseURL, cfg.APIKey, cfg.Model)

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
