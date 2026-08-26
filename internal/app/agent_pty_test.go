//go:build !windows

package toudocu

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestAgentPTY(t *testing.T) {
	output := make(chan string, 8)
	terminal := newAgentPTY(func(event agentTerminalEvent) {
		if event.Type == "output" {
			bytes, _ := base64.StdEncoding.DecodeString(event.Data)
			output <- string(bytes)
		}
	})
	if err := terminal.startCommand(t.TempDir(), "sh", []string{"-c", "read value; printf 'received:%s\\n' \"$value\"; sleep 30"}); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Resize(100, 30); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Write("hello\n"); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	combined := ""
	for !strings.Contains(combined, "received:hello") {
		select {
		case chunk := <-output:
			combined += chunk
		case <-deadline:
			t.Fatalf("PTY output=%q", combined)
		}
	}
	if err := terminal.Interrupt(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := terminal.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAgentPTYFullAccess(t *testing.T) {
	if args := terminalCodexArgs(AgentLaunchDefault); len(args) != 0 {
		t.Fatalf("default args=%v", args)
	}
	if args := terminalCodexArgs(AgentLaunchFullAccess); len(args) != 1 || args[0] != "--yolo" {
		t.Fatalf("full-access args=%v", args)
	}
}
