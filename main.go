package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/agenttea/internal/api"
	"github.com/user/agenttea/internal/config"
	"github.com/user/agenttea/internal/logger"
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

	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v (不影响运行)\n", err)
	}
	logger.Info("AgentTea v%s 启动", api.Version)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("配置加载失败: %v", err)
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}
	logger.Info("配置加载成功, 模型: %s, BaseURL: %s", cfg.Model, cfg.BaseURL)

	if *modelName != "" {
		cfg.Model = *modelName
		logger.Info("命令行指定模型: %s", *modelName)
	}

	if cfg.APIKey == "" {
		logger.Error("API 密钥未设置")
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

	logger.Info("应用启动")
	if _, err := p.Run(); err != nil {
		logger.Error("运行失败: %v", err)
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}
	logger.Info("应用正常退出")
}
