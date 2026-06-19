package main

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func main() {
	// Check if we're being invoked as daemon supervisor
	if os.Getenv("_ATOMCODE_DAEMON_SUPERVISOR") == "1" {
		port, _ := strconv.Atoi(os.Getenv("_ATOMCODE_DAEMON_PORT"))
		if port == 0 {
			port = 13457
		}
		RunSupervisor(port)
		return
	}

	rootCmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands:"},
		&cobra.Group{ID: "service", Title: "Service Management:"},
		&cobra.Group{ID: "query", Title: "Query & Info:"},
	)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}