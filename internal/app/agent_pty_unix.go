//go:build !windows

package toudocu

import "os"

func agentPTYSupported() bool { return true }

func defaultProjectShell() (string, []string) {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell, nil
	}
	return "/bin/sh", nil
}
