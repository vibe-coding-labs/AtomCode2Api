package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/atmc"
)

var chatModel string
var chatStream bool
var chatMaxTokens int

var chatCmd = &cobra.Command{
	Use:     "chat [message]",
	Short:   "发送聊天消息",
	Long:    "通过 AtomCode Daemon 发送一条聊天消息并返回响应。",
	GroupID: "core",
	Example: `  atomcode-proxy chat "你好"
  atomcode-proxy chat -m deepseek-v4-flash "写一段代码"
  atomcode-proxy chat -s "解释量子计算"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getDaemonClient()
		providers, _ := client.ListProviders()
		provider := atmc.FindProviderForModel(providers, chatModel)

		req := &atmc.ChatRequest{
			Message:  fmt.Sprintf("User: %s", args[0]),
			Provider: provider,
			Stream:   chatStream,
		}

		if chatStream {
			ch, err := client.ChatStream(req)
			if err != nil {
				return err
			}
			for ev := range ch {
				switch ev.Type {
				case "text":
					fmt.Print(ev.Content)
				case "done":
					fmt.Println()
					return nil
				case "error":
					fmt.Printf("\nError: %s\n", ev.Message)
					return nil
				}
			}
			return nil
		}

		// Non-streaming — collect all events
		ch, err := client.ChatStream(req)
		if err != nil {
			return err
		}
		var fullText string
		for ev := range ch {
			switch ev.Type {
			case "text":
				fullText += ev.Content
			case "done":
				fmt.Println(fullText)
				return nil
			case "error":
				return fmt.Errorf("daemon error: %s", ev.Message)
			}
		}
		fmt.Println(fullText)
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "deepseek-v4-flash", "模型名称")
	chatCmd.Flags().BoolVarP(&chatStream, "stream", "s", false, "流式输出")
	chatCmd.Flags().IntVar(&chatMaxTokens, "max-tokens", 64000, "最大输出 token 数")
	rootCmd.AddCommand(chatCmd)
}
