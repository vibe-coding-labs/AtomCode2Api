package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibe-coding-labs/AtomCode2API/pkg/atmc"
)

var setupCmd = &cobra.Command{
	Use:     "setup",
	Short:   "登录 + CodingPlan 领取",
	Long:    "执行 OAuth 登录并领取 CodingPlan 免费额度。需要 AtomCode daemon 已在运行。",
	GroupID: "core",
	Example: `  # 完整设置（登录 + 领取额度）
  atomcode-2api setup

  # 指定 daemon 地址
  ATOMCODE_DAEMON_URL=http://localhost:13456 atomcode-2api setup`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetup()
	},
}

var loginCmd = &cobra.Command{
	Use:     "login",
	Short:   "仅 OAuth 登录",
	Long:    "执行 OAuth 登录。需要 AtomCode daemon 已在运行。",
	GroupID: "core",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogin()
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(loginCmd)
}

func getDaemonClient() *atmc.Client {
	daemonURL := getEnvDefault("ATOMCODE_DAEMON_URL", "http://localhost:13456")
	return atmc.NewClient(daemonURL)
}

func runLogin() error {
	client := getDaemonClient()
	return interactiveLogin(client)
}

func runSetup() error {
	client := getDaemonClient()

	// Login first
	if err := interactiveLogin(client); err != nil {
		return err
	}

	// Claim CodingPlan
	fmt.Println()
	fmt.Println("Claiming CodingPlan...")
	cp, err := client.CodingPlanSetup()
	if err != nil {
		return fmt.Errorf("codingplan setup failed: %w", err)
	}
	if cp.Success {
		fmt.Println("✅ CodingPlan claimed!")
		fmt.Printf("   Default provider: %s\n", cp.DefaultProvider)
		for _, p := range cp.Providers {
			fmt.Printf("   - %s (%s)\n", p.Name, p.Model)
		}
	} else {
		fmt.Printf("CodingPlan setup returned: %+v\n", cp)
	}
	return nil
}

func interactiveLogin(client *atmc.Client) error {
	// Check if already logged in
	auth, err := client.AuthStatus()
	if err != nil {
		return fmt.Errorf("cannot check auth status: %w", err)
	}
	if auth.LoggedIn && auth.User != nil {
		fmt.Printf("Already logged in as %s (%s)\n", auth.User.Name, auth.User.Username)
		return nil
	}

	loginData, err := client.LoginStart()
	if err != nil {
		return fmt.Errorf("cannot start login: %w", err)
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("  Open this URL in your browser to authorize:")
	fmt.Println("  " + loginData.URL)
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println()
	fmt.Printf("Login session expires in %d seconds.\n", loginData.ExpiresInSeconds)

	deadline := time.Now().Add(time.Duration(loginData.ExpiresInSeconds) * time.Second)
	pollInterval := 2 * time.Second

	reader := bufio.NewReader(os.Stdin)

	for time.Now().Before(deadline) {
		result, err := client.LoginPoll(loginData.LoginID)
		if err != nil {
			fmt.Printf("Poll error: %v\n", err)
			time.Sleep(pollInterval)
			continue
		}
		if result.Status == "authorized" && result.User != nil {
			fmt.Println()
			fmt.Printf("✅ Login successful as %s (%s)!\n", result.User.Name, result.User.Username)
			return nil
		}
		if result.Error != "" {
			fmt.Printf("Login error: %s\n", result.Error)
			return fmt.Errorf("login failed: %s", result.Error)
		}
		fmt.Print(".")
		reader.ReadString('\n') // wait for Enter key
		// Also poll on timer
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("login timed out")
}
