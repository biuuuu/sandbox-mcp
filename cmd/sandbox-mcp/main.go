package main

import (
	"context"
	"flag"
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/pottekkat/sandbox-mcp/internal/appconfig"
	"github.com/pottekkat/sandbox-mcp/internal/config"
	"github.com/pottekkat/sandbox-mcp/internal/sandbox"
)

func main() {
	// 解析命令行参数
	stdio := flag.Bool("stdio", false, "通过标准输入输出启动MCP")
	sse := flag.Bool("sse", false, "通过SSE启动MCP")
	build := flag.Bool("build", false, "为所有沙箱构建Docker镜像")
	pull := flag.Bool( "pull", false, "从GitHub拉取默认沙箱")
	force := flag.Bool("force", false, "拉取时强制覆盖现有沙箱")
	flag.Parse()

	// 配置日志记录
	// TODO: 根据MCP规范改进日志记录
	log.SetPrefix("[沙箱 MCP] ")
	log.SetFlags(log.Ldate | log.Ltime)

	// 加载应用程序配置
	cfg, err := appconfig.LoadConfig()
	if err != nil {
		log.Fatalf("加载sandbox-mcp配置失败: %v", err)
	}

	// 如果存在pull标志，则拉取沙箱
	if *pull {
		if err := sandbox.PullSandboxes(cfg.SandboxesPath, *force); err != nil {
			log.Fatalf("拉取沙箱失败: %v", err)
		}
		return
	}

	// 从配置路径加载沙箱配置
	configs, err := config.LoadSandboxConfigs(cfg.SandboxesPath)
	if err != nil {
		log.Fatalf("加载沙箱配置失败: %v", err)
	}

	// 如果存在build标志，则构建Docker镜像
	if *build {
		log.Println("正在为所有沙箱构建Docker镜像...")
		for _, sandboxCfg := range configs {
			if err := sandbox.BuildImage(context.Background(), sandboxCfg, cfg.SandboxesPath); err != nil {
				log.Printf("为沙箱 %s 构建镜像失败: %v", sandboxCfg.Id, err)
				continue
			}
		}
		return
	}

	// 仅当存在sse标志时启动MCP服务器
	if *sse {
		// 创建新的MCP服务器
		s := server.NewMCPServer(
			"Sandbox MCP",
			"0.1.0",
			// 当工具列表更改时，我们不通知
			// 目前工具列表不会更改
			server.WithToolCapabilities(false),
		)
		sseServer := server.NewSSEServer(s)
		// 为每个沙箱配置创建并添加工具
		for _, cfg := range configs {
			// 从配置创建新工具
			tool := sandbox.NewSandboxTool(cfg)

			// 使用沙箱配置创建处理程序
			handler := sandbox.NewSandboxToolHandler(cfg)

			// 将工具添加到服务器
			s.AddTool(tool, handler)

			log.Printf("从配置添加 %s 工具", cfg.Id)
		}

		log.Println("启动沙箱MCP服务器...")
		log.Printf("沙箱MCP服务器已启动，地址: %s", "localhost:10061")
		if err := sseServer.Start("0.0.0.0:10061"); err != nil {
			log.Printf("启动服务器错误: %v", err)
			return
		}
	}
	// 仅当存在stdio标志时启动MCP服务器
	if *stdio {
		// 创建新的MCP服务器
		s := server.NewMCPServer(
			"Sandbox MCP",
			"0.1.0",
			// 当工具列表更改时，我们不通知
			// 目前工具列表不会更改
			server.WithToolCapabilities(false),
		)

		// 为每个沙箱配置创建并添加工具
		for _, cfg := range configs {
			// 从配置创建新工具
			tool := sandbox.NewSandboxTool(cfg)

			// 使用沙箱配置创建处理程序
			handler := sandbox.NewSandboxToolHandler(cfg)

			// 将工具添加到服务器
			s.AddTool(tool, handler)

			log.Printf("从配置添加 %s 工具", cfg.Id)
		}

		log.Println("启动沙箱MCP服务器...")

		// 启动服务器
		if err := server.ServeStdio(s); err != nil {
			log.Printf("启动服务器错误: %v\n", err)
		}
	}
}
