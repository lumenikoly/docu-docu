package toudocu_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestReleasedBinaryWithoutNodeRuntime(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(docs, "architecture"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, ".toudocu", "config.yml"):      "documentationVersion: 2\nproject:\n  locale: en\n",
		filepath.Join(docs, "index.md"):                    "# Release fixture\n\nA minimal project.\n",
		filepath.Join(docs, "architecture", "overview.md"): "# Architecture\n\nThe release fixture has one component.\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	binary := filepath.Join(root, "toudocu")
	command := exec.Command("go", "build", "-o", binary, "./cmd/toudocu")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build release binary: %v\n%s", err, output)
	}
	environment := append(os.Environ(), "PATH="+filepath.Join(root, "empty-path"))
	run := func(args ...string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = environment
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("toudocu %v: %v\n%s", args, err, output)
		}
	}
	run("check", docs, "--repository-root", root)
	run("build", docs, "-o", filepath.Join(root, "site"), "--repository-root", root)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	serve := exec.CommandContext(ctx, binary, "serve", docs, "-o", filepath.Join(root, "serve"), "--repository-root", root, "--host", "127.0.0.1", "--port", fmt.Sprint(port), "--no-update-check")
	serve.Env = environment
	if err := serve.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		serve.Wait()
	}()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(25 * time.Millisecond) {
		response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("released binary did not start serve without Node.js")
		}
	}
}
