package main

import (
	"ava/internal/tts"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 初始化 Speaker 管理器
	speakerManager := tts.GetGlobalManager()
	speakerManager.SetConfig(&tts.ToolSpeakerConfig{
		VoiceType:  "zh_male_lengkugege_emo_v2_mars_bigtts",
		ResourceID: "seed-tts-1.0",
		AccessKey:  "n1uNFm540_2oItTs0UsULkWWvuzQiXbD",
		AppKey:     "5711022755",
		Encoding:   "pcm",
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
		SpeedRatio: 1.1,
	})

	// 确保程序退出时清理资源
	defer func() {
		if err := speakerManager.Close(); err != nil {
			log.Printf("Failed to close speaker manager: %v", err)
		}
	}()

	// 创建 DuckDuckGo 工具
	// searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	// if err != nil {
	// 	log.Fatalf("NewTextSearchTool failed, err=%v", err)
	// 	return
	// }

	// 创建 Speech Tools（不包括 say_text，因为它会在响应后自动调用）
	pauseSpeechTool := NewPauseSpeechTool(speakerManager)
	resumeSpeechTool := NewResumeSpeechTool(speakerManager)
	stopSpeechTool := NewStopSpeechTool(speakerManager)

	// 初始化 tools
	todoTools := []tool.BaseTool{
		&ListTodoTool{}, // 实现Tool接口
		pauseSpeechTool,
		resumeSpeechTool,
		stopSpeechTool,
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

	// 创建一个只包含 ChatModel 的 chain（用于手动处理逻辑）
	// ChatModel 输出 *schema.Message，所以 chain 类型应该是 []*schema.Message -> *schema.Message
	chain := compose.NewChain[[]*schema.Message, *schema.Message]()
	chain.AppendChatModel(chatModel, compose.WithNodeName("chat_model"))
	chatModelAgent, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 创建读取器
	reader := bufio.NewReader(os.Stdin)

	// 保存对话历史，首先添加系统提示词
	systemPrompt := `你是一个智能语音助手。你的所有回复都会自动通过语音播放给用户，所以你只需要正常回复文本内容即可。

你可以使用以下工具来控制语音播放：
- pause_speech: 暂停当前播放
- resume_speech: 恢复播放
- stop_speech: 停止播放

注意：你不需要手动调用语音合成功能，系统会自动将你的文本回复转换为语音播放。`

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: systemPrompt,
		},
	}

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

		// 手动实现 agent 循环逻辑
		fmt.Print("Agent: ")
		resp, err := invokeAgent(ctx, chatModelAgent, todoToolsNode, messages)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			// 移除最后一条用户消息，因为处理失败
			messages = messages[:len(messages)-1]
			continue
		}

		// 输出结果并更新消息历史，同时自动播放语音
		var assistantContent strings.Builder
		for _, msg := range resp {
			// 只处理 Assistant 角色的消息内容
			if msg.Role == schema.Assistant && msg.Content != "" {
				fmt.Println(msg.Content)
				assistantContent.WriteString(msg.Content)

				// 将 agent 的响应添加到历史中
				messages = append(messages, msg)
			} else {
				// 其他类型的消息（如 tool call 结果）也添加到历史
				messages = append(messages, msg)
			}
			log.Printf("msg: %+v", msg)
		}

		// 自动播放 Assistant 的回复内容
		if assistantContent.Len() > 0 {
			text := assistantContent.String()
			speaker, err := speakerManager.GetSpeaker(ctx)
			if err != nil {
				log.Printf("获取 Speaker 失败，跳过语音播放: %v", err)
			} else {
				// 播放文本，end=true 表示这是完整的回复
				// 注意：不要调用 Stop()，因为这会结束 session，导致后续调用失败
				// end=true 只是标记文本结束，但 session 应该保持运行以便后续对话
				if err := speaker.Say(text, false); err != nil {
					log.Printf("语音播放失败: %v", err)
				}
			}
		}

		fmt.Println()
	}
}

// invokeAgent 手动实现 agent 循环逻辑：ChatModel -> ToolsNode (如果有 tool call) -> ChatModel
func invokeAgent(ctx context.Context, chatModelAgent compose.Runnable[[]*schema.Message, *schema.Message], toolsNode *compose.ToolsNode, messages []*schema.Message) ([]*schema.Message, error) {
	maxIterations := 10 // 防止无限循环
	for i := 0; i < maxIterations; i++ {
		// 调用 ChatModel（返回单个消息）
		chatMsg, err := chatModelAgent.Invoke(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("ChatModel invoke failed: %w", err)
		}

		// 检查是否有 tool call
		hasToolCall := chatMsg.Role == schema.Assistant && len(chatMsg.ToolCalls) > 0

		// 如果没有 tool call，直接返回
		if !hasToolCall {
			return []*schema.Message{chatMsg}, nil
		}

		// 有 tool call，调用 ToolsNode
		toolResp, err := toolsNode.Invoke(ctx, chatMsg)
		if err != nil {
			return nil, fmt.Errorf("ToolsNode invoke failed: %w", err)
		}

		// 将 ChatModel 和 ToolsNode 的响应都添加到消息历史
		messages = append(messages, chatMsg)
		messages = append(messages, toolResp...)
		// 继续循环，让 ChatModel 处理 tool 的结果
	}

	return nil, fmt.Errorf("max iterations reached")
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
