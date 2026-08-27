//go:build windows

package toudocu

import "os"

func agentPTYSupported() bool { return true }

func defaultProjectShell() (string, []string) {
	if shell := os.Getenv("COMSPEC"); shell != "" {
		return shell, nil
	}
	return "cmd.exe", nil
}
