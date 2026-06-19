package omni

import "runtime"

// ServiceName returns the human-readable service name.
func ServiceName() string {
	return "atomcode-proxy"
}

// SupportsSystemd returns true if the platform supports systemd.
func SupportsSystemd() bool {
	return runtime.GOOS == "linux"
}

// SupportsLaunchd returns true if the platform supports launchd.
func SupportsLaunchd() bool {
	return runtime.GOOS == "darwin"
}
