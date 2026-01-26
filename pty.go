package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PTYSession represents an active PTY
type PTYSession struct {
	id     string
	cmd    *exec.Cmd
	pty    *os.File
	cancel chan struct{}
}

// PTYManager manages multiple PTY sessions
type PTYManager struct {
	ctx      context.Context
	sessions map[string]*PTYSession
	mu       sync.RWMutex
}

func NewPTYManager(ctx context.Context) *PTYManager {
	return &PTYManager{
		ctx:      ctx,
		sessions: make(map[string]*PTYSession),
	}
}

// StartTerminalListeners registers event handlers for terminal operations
func (a *App) StartTerminalListeners() {
	if a.ptyManager == nil {
		a.ptyManager = NewPTYManager(a.ctx)
	}

	runtime.EventsOn(a.ctx, "terminal:start", func(data ...interface{}) {
		params, ok := firstAs[map[string]interface{}](data)
		if !ok {
			slog.Error("terminal:start invalid params")
			return
		}
		id := mapStr(params, "id")
		cols := mapInt(params, "cols")
		rows := mapInt(params, "rows")
		if cols == 0 {
			cols = 80
		}
		if rows == 0 {
			rows = 24
		}
		slog.Info("terminal:start", "id", id, "cols", cols, "rows", rows)
		if err := a.ptyManager.Start(id, uint16(cols), uint16(rows)); err != nil {
			slog.Error("terminal start failed", "id", id, "error", err)
		}
	})

	runtime.EventsOn(a.ctx, "terminal:input", func(data ...interface{}) {
		params, ok := firstAs[map[string]interface{}](data)
		if !ok {
			slog.Error("terminal:input invalid params")
			return
		}
		id := mapStr(params, "id")
		input := mapStr(params, "data")
		slog.Debug("terminal:input", "id", id, "len", len(input))
		a.ptyManager.Write(id, []byte(input))
	})

	runtime.EventsOn(a.ctx, "terminal:resize", func(data ...interface{}) {
		params, ok := firstAs[map[string]interface{}](data)
		if !ok {
			return
		}
		id := mapStr(params, "id")
		cols := mapInt(params, "cols")
		rows := mapInt(params, "rows")
		a.ptyManager.Resize(id, uint16(cols), uint16(rows))
	})

	runtime.EventsOn(a.ctx, "terminal:stop", func(data ...interface{}) {
		params, ok := firstAs[map[string]interface{}](data)
		if !ok {
			return
		}
		id := mapStr(params, "id")
		a.ptyManager.Stop(id)
	})
}

// Start creates a new PTY session
func (m *PTYManager) Start(id string, cols, rows uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing session if any
	if s, ok := m.sessions[id]; ok {
		select {
		case <-s.cancel:
			// Already closed
		default:
			close(s.cancel)
		}
		s.pty.Close()
		s.cmd.Process.Kill()
		s.cmd.Wait()
		delete(m.sessions, id)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	cmd.Env = os.Environ()

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return err
	}

	session := &PTYSession{
		id:     id,
		cmd:    cmd,
		pty:    ptmx,
		cancel: make(chan struct{}),
	}
	m.sessions[id] = session

	// Read loop - emit output to frontend
	go m.readLoop(session)

	return nil
}

func (m *PTYManager) readLoop(session *PTYSession) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-session.cancel:
			return
		default:
			n, err := session.pty.Read(buf)
			if err != nil {
				if err != io.EOF {
					// PTY closed or error
				}
				return
			}
			if n > 0 {
				runtime.EventsEmit(m.ctx, "terminal:"+session.id+":output", string(buf[:n]))
			}
		}
	}
}

// Write sends input to a PTY session
func (m *PTYManager) Write(id string, data []byte) {
	m.mu.RLock()
	s := m.sessions[id]
	m.mu.RUnlock()
	if s != nil {
		s.pty.Write(data)
	}
}

// Resize changes the PTY window size
func (m *PTYManager) Resize(id string, cols, rows uint16) {
	m.mu.RLock()
	s := m.sessions[id]
	m.mu.RUnlock()
	if s != nil {
		pty.Setsize(s.pty, &pty.Winsize{Cols: cols, Rows: rows})
	}
}

// Stop terminates a PTY session
func (m *PTYManager) Stop(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		select {
		case <-s.cancel:
		default:
			close(s.cancel)
		}
		s.pty.Close()
		s.cmd.Process.Kill()
		s.cmd.Wait()
		delete(m.sessions, id)
	}
}

// StopAll terminates all PTY sessions
func (m *PTYManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		select {
		case <-s.cancel:
		default:
			close(s.cancel)
		}
		s.pty.Close()
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
	m.sessions = make(map[string]*PTYSession)
}
