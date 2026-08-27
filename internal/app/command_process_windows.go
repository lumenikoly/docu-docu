//go:build windows

package toudocu

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func newShellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd.exe", "/S", "/C", command)
	configureProcessTree(cmd)
	return cmd
}

func configureProcessTree(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return killProcessTree(cmd)
	}
	cmd.WaitDelay = 2 * time.Second
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	killTree := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	if err := killTree.Run(); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}
