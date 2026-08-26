//go:build !windows

package toudocu

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOSCommandRunnerTimeoutKillsProcessGroup(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "child-survived")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	command := `(sleep 1; printf survived > "` + marker + `") & wait`
	exitCode, err := (osCommandRunner{}).Run(ctx, command, root, io.Discard, io.Discard)
	if err == nil || exitCode >= 0 {
		t.Fatalf("expected cancelled command, exit=%d err=%v", exitCode, err)
	}
	if elapsed := time.Since(started); elapsed > 900*time.Millisecond {
		t.Fatalf("timeout waited for a descendant process: %s", elapsed)
	}

	time.Sleep(1100 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("descendant process survived timeout: %v", err)
	}
}

func TestAgentProcessTree(t *testing.T) { TestOSCommandRunnerTimeoutKillsProcessGroup(t) }
