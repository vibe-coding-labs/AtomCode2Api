package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "显示当前登录用户",
	Long:    "显示 AtomCode Daemon 的当前登录用户信息。",
	GroupID: "query",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getDaemonClient()
		auth, err := client.AuthStatus()
		if err != nil {
			return fmt.Errorf("daemon unreachable: %w", err)
		}
		if !auth.LoggedIn || auth.User == nil {
			fmt.Println("Not logged in.")
			return nil
		}
		fmt.Printf("Logged in as: %s (%s)\n", auth.User.Name, auth.User.Username)
		if auth.Token != nil {
			expiresH := auth.Token.ExpiresIn / 3600
			fmt.Printf("Token expires: %dh, has refresh: %t\n", expiresH, auth.Token.HasRefreshToken)
		}
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "检查 daemon 和登录状态",
	Long:    "全面检查 daemon 健康、登录、CodingPlan 状态。",
	GroupID: "query",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := getDaemonClient()
		daemonURL := getEnvDefault("ATOMCODE_DAEMON_URL", "http://localhost:13456")

		fmt.Println("AtomCode 2API — Health Check")
		fmt.Println(strings.Repeat("═", 40))

		h, err := client.Health()
		if err != nil {
			fmt.Printf("✗ Daemon: unreachable (%v)\n", err)
			fmt.Printf("  URL: %s\n", daemonURL)
			return nil
		}
		fmt.Printf("✓ Daemon: %s (v%s)\n", h.Status, h.Version)

		auth, err := client.AuthStatus()
		if err == nil && auth.LoggedIn && auth.User != nil {
			fmt.Printf("✓ Auth: %s (%s)\n", auth.User.Name, auth.User.Username)
		} else {
			fmt.Printf("✗ Auth: not logged in\n")
		}

		models, err := client.ListModels()
		if err == nil {
			fmt.Printf("✓ Models: %d available\n", len(models))
		} else {
			fmt.Printf("✗ Models: %v\n", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(doctorCmd)
}
