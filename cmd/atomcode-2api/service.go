package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "管理系统服务",
	Long:    "安装/卸载系统服务（systemd / launchd），实现开机自启。",
	GroupID: "service",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装系统服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		return installService()
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "卸载系统服务",
	RunE: func(cmd *cobra.Command, args []string) error {
		return uninstallService()
	},
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd, serviceUninstallCmd)
	rootCmd.AddCommand(serviceCmd)
}

func installService() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return installSystemd(bin)
	case "darwin":
		return installLaunchd(bin)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func uninstallService() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func installSystemd(bin string) error {
	home, _ := os.UserHomeDir()
	svc := fmt.Sprintf(`[Unit]
Description=AtomCode 2API
After=network.target

[Service]
Type=simple
ExecStart=%s serve
Restart=on-failure
RestartSec=5
Environment=ATOMCODE_TELEMETRY=0

[Install]
WantedBy=default.target
`, bin)
	dir := home + "/.config/systemd/user"
	os.MkdirAll(dir, 0755)
	path := dir + "/atomcode-2api.service"
	if err := os.WriteFile(path, []byte(svc), 0644); err != nil {
		return err
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	log.Printf("Systemd service installed: %s", path)
	log.Printf("Run: systemctl --user enable --now atomcode-2api")
	return nil
}

func installLaunchd(bin string) error {
	home, _ := os.UserHomeDir()
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.atomcode.proxy</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>serve</string>
	</array>
	<key>KeepAlive</key>
	<true/>
	<key>RunAtLoad</key>
	<true/>
	<key>EnvironmentVariables</key>
	<dict>
		<key>ATOMCODE_TELEMETRY</key>
		<string>0</string>
	</dict>
	<key>StandardOutPath</key>
	<string>%s/Library/Logs/atomcode-2api.log</string>
	<key>StandardErrorPath</key>
	<string>%s/Library/Logs/atomcode-2api.log</string>
</dict>
</plist>
`, bin, home, home)
	dir := home + "/Library/LaunchAgents"
	os.MkdirAll(dir, 0755)
	path := dir + "/com.atomcode.proxy.plist"
	if err := os.WriteFile(path, []byte(plist), 0644); err != nil {
		return err
	}
	exec.Command("launchctl", "load", path).Run()
	log.Printf("Launchd service installed: %s", path)
	return nil
}

func uninstallSystemd() error {
	exec.Command("systemctl", "--user", "stop", "atomcode-2api").Run()
	exec.Command("systemctl", "--user", "disable", "atomcode-2api").Run()
	home, _ := os.UserHomeDir()
	os.Remove(home + "/.config/systemd/user/atomcode-2api.service")
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	log.Printf("Systemd service uninstalled")
	return nil
}

func uninstallLaunchd() error {
	home, _ := os.UserHomeDir()
	path := home + "/Library/LaunchAgents/com.atomcode.proxy.plist"
	exec.Command("launchctl", "unload", path).Run()
	os.Remove(path)
	log.Printf("Launchd service uninstalled")
	return nil
}
