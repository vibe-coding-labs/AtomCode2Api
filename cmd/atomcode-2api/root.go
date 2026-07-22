package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose        bool
	skipValidation bool
)

var rootCmd = &cobra.Command{
	Use:     "atomcode-2api",
	Short:   "AtomCode 2API — OpenAI/Anthropic 兼容代理",
	Long: `将 AtomCode Daemon 的 CodingPlan 免费额度以 OpenAI/Anthropic 兼容 API 形式暴露。

支持：
  - OpenAI Chat Completions API (POST /v1/chat/completions)
  - Anthropic Messages API (POST /v1/messages)
  - 流式/非流式输出
  - 多轮对话上下文保持
  - 守护进程模式（崩溃自动重启）`,
	Version: Version,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "启用详细日志")
	rootCmd.PersistentFlags().BoolVar(&skipValidation, "skip-validation", false, "跳过凭据验证")
}

// Execute is called by main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}