package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Short:   "列出可用模型",
	Long:    "从 AtomCode Daemon 获取可用的模型列表。",
	GroupID: "query",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getDaemonClient()
		models, err := client.ListModels()
		if err != nil {
			return fmt.Errorf("fetch models: %w", err)
		}
		if len(models) == 0 {
			fmt.Println("No models available.")
			return nil
		}
		fmt.Printf("Available models (%d):\n", len(models))
		for _, m := range models {
			fmt.Printf("  - %s", m.ID)
			if m.Provider != "" {
				fmt.Printf(" (provider: %s)", m.Provider)
			}
			fmt.Println()
		}
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "查看代理和 daemon 状态",
	Long:    "查看代理服务器、Daemon 连接状态和登录信息。",
	GroupID: "query",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getDaemonClient()

		fmt.Println("AtomCode Proxy — Status")
		fmt.Println("=" + "================================")

		// Daemon health
		h, err := client.Health()
		if err != nil {
			fmt.Printf("Daemon:       ❌ unreachable (%v)\n", err)
		} else {
			fmt.Printf("Daemon:       ✅ %s (v%s)\n", h.Status, h.Version)
		}

		// Auth status
		auth, authErr := client.AuthStatus()
		if authErr == nil && auth.LoggedIn && auth.User != nil {
			fmt.Printf("Logged in:    ✅ %s (%s)\n", auth.User.Name, auth.User.Username)
			if auth.Token != nil {
				expiresH := auth.Token.ExpiresIn / 3600
				fmt.Printf("Token:        expires in %dh, refresh: %t\n",
					expiresH, auth.Token.HasRefreshToken)
			}
		} else {
			fmt.Printf("Logged in:    ❌\n")
		}

		// Models
		models, err := client.ListModels()
		if err == nil {
			fmt.Printf("Models:       %d available\n", len(models))
		}

		daemonURL := getEnvDefault("ATOMCODE_DAEMON_URL", "http://localhost:13456")
		fmt.Printf("Daemon URL:   %s\n", daemonURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(statusCmd)
}
