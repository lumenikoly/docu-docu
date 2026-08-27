package toudocu

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	ptylib "github.com/aymanbagabas/go-pty"
)

type agentTerminalState struct {
	Available bool   `json:"available"`
	Active    bool   `json:"active"`
	Failure   string `json:"failure,omitempty"`
}

type agentTerminalEvent struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
}

type agentPTY struct {
	mu      sync.Mutex
	pty     ptylib.Pty
	cmd     *ptylib.Cmd
	done    chan struct{}
	active  bool
	failure string
	publish func(agentTerminalEvent)
}

func newAgentPTY(publish func(agentTerminalEvent)) *agentPTY { return &agentPTY{publish: publish} }

func (p *agentPTY) Snapshot(available bool) agentTerminalState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return agentTerminalState{Available: available && agentPTYSupported(), Active: p.active, Failure: p.failure}
}

func (p *agentPTY) StartShell(cwd string) error {
	executable, args := defaultProjectShell()
	return p.startCommand(cwd, executable, args)
}

func (p *agentPTY) startCommand(cwd, executable string, args []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !agentPTYSupported() {
		return errors.New("project terminal is unsupported on this platform")
	}
	if p.active {
		return errors.New("project terminal is already active")
	}
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve terminal cwd: %w", err)
	}
	terminal, err := ptylib.New()
	if err != nil {
		return fmt.Errorf("create agent PTY: %w", err)
	}
	cmd := terminal.Command(executable, args...)
	cmd.Dir = absoluteCWD
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if err = cmd.Start(); err != nil {
		_ = terminal.Close()
		return fmt.Errorf("start project terminal: %w", err)
	}
	p.pty, p.cmd, p.done, p.active, p.failure = terminal, cmd, make(chan struct{}), true, ""
	go p.read(terminal, cmd, p.done)
	return nil
}

func (p *agentPTY) read(terminal ptylib.Pty, cmd *ptylib.Cmd, done chan struct{}) {
	buffer := make([]byte, 32<<10)
	for {
		n, err := terminal.Read(buffer)
		if n > 0 {
			p.publish(agentTerminalEvent{Type: "output", Data: base64.StdEncoding.EncodeToString(buffer[:n])})
		}
		if err != nil {
			break
		}
	}
	err := cmd.Wait()
	p.mu.Lock()
	if p.pty == terminal {
		wasActive := p.active
		p.active = false
		if wasActive && err != nil && !errors.Is(err, os.ErrProcessDone) {
			p.failure = err.Error()
		}
	}
	p.mu.Unlock()
	_ = terminal.Close()
	close(done)
	p.publish(agentTerminalEvent{Type: "exit"})
}

func (p *agentPTY) Write(value string) error {
	p.mu.Lock()
	terminal, active := p.pty, p.active
	p.mu.Unlock()
	if !active || terminal == nil {
		return errors.New("project terminal is not active")
	}
	_, err := io.WriteString(terminal, value)
	return err
}

func (p *agentPTY) Resize(columns, rows int) error {
	if columns < 2 || columns > 500 || rows < 2 || rows > 500 {
		return errors.New("terminal size is outside 2..500")
	}
	p.mu.Lock()
	terminal, active := p.pty, p.active
	p.mu.Unlock()
	if !active || terminal == nil {
		return errors.New("project terminal is not active")
	}
	return terminal.Resize(columns, rows)
}

func (p *agentPTY) Interrupt() error { return p.Write("\x03") }

func (p *agentPTY) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.active || p.pty == nil {
		p.mu.Unlock()
		return nil
	}
	terminal, cmd, done := p.pty, p.cmd, p.done
	p.active = false
	p.mu.Unlock()
	_ = terminal.Close()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ErrAgentStopUnconfirmed
	}
}
