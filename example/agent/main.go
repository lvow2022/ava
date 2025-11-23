package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	duckduckgo "github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	// 创建 DuckDuckGo 工具
	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		log.Fatalf("NewTextSearchTool failed, err=%v", err)
		return
	}
	// 初始化 tools
	todoTools := []tool.BaseTool{
		&ListTodoTool{}, // 实现Tool接口
		searchTool,
	}

	// 创建并配置 ChatModel
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		Model:   "doubao-1-5-pro-32k-250115",
		APIKey:  "42189dbf-a896-4162-b9d8-3dccf4b1ded5",
	})
	if err != nil {
		log.Fatal(err)
	}
	// 获取工具信息并绑定到 ChatModel
	toolInfos := make([]*schema.ToolInfo, 0, len(todoTools))
	for _, tool := range todoTools {
		info, err := tool.Info(ctx)
		if err != nil {
			log.Fatal(err)
		}
		toolInfos = append(toolInfos, info)
	}
	err = chatModel.BindTools(toolInfos)
	if err != nil {
		log.Fatal(err)
	}

	// 创建 tools 节点
	todoToolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
		Tools: todoTools,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 构建完整的处理链
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.
		AppendChatModel(chatModel, compose.WithNodeName("chat_model")).
		AppendToolsNode(todoToolsNode, compose.WithNodeName("tools"))

	// 编译并运行 chain
	agent, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 创建读取器
	reader := bufio.NewReader(os.Stdin)

	// 保存对话历史
	messages := make([]*schema.Message, 0)

	fmt.Println("=== Agent 交互式对话 ===")
	fmt.Println("输入您的问题（输入 'exit' 或 'quit' 退出）")
	fmt.Println()

	// 交互式循环
	for {
		fmt.Print("您: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("读取输入失败: %v", err)
			continue
		}

		// 去除换行符和空格
		input = strings.TrimSpace(input)

		// 检查退出命令
		if input == "exit" || input == "quit" || input == "q" {
			fmt.Println("再见！")
			break
		}

		// 忽略空输入
		if input == "" {
			continue
		}

		// 添加用户消息到历史
		userMsg := &schema.Message{
			Role:    schema.User,
			Content: input,
		}
		messages = append(messages, userMsg)

		// 调用 agent
		fmt.Print("Agent: ")
		resp, err := agent.Invoke(ctx, messages)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			// 移除最后一条用户消息，因为处理失败
			messages = messages[:len(messages)-1]
			continue
		}

		// 输出结果并更新消息历史
		for _, msg := range resp {
			fmt.Println(msg.Content)
			// 将 agent 的响应也添加到历史中
			messages = append(messages, msg)
		}
		fmt.Println()
	}
}

type ListTodoTool struct{}

func (lt *ListTodoTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_todo",
		Desc: "List all todo items",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"finished": {
				Desc:     "filter todo items if finished",
				Type:     schema.Boolean,
				Required: false,
			},
		}),
	}, nil
}

func (lt *ListTodoTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Mock调用逻辑
	return `{"todos": [{"id": "1", "content": "在2024年12月10日之前完成Eino项目演示文稿的准备工作", "started_at": 1717401600, "deadline": 1717488000, "done": false}]}`, nil
}
