package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vibe-coding-labs/AtomCodeProxy/pkg/atmc"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "执行网页搜索",
	Long:  "通过 AtomCode Daemon 执行网页搜索。",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getDaemonClient()
		req := &atmc.ChatRequest{
			Message:  fmt.Sprintf("User: %s", args[0]),
			Provider: "deepseek",
		}
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
				return fmt.Errorf("error: %s", ev.Message)
			}
		}
		fmt.Println(fullText)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
