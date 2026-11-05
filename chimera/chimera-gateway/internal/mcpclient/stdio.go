package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
)

// StdioConfig launches a child process for MCP stdio transport.
type StdioConfig struct {
	Command string
	Args    []string
	Env     []string
}

// stdioTransport exchanges line-framed JSON-RPC on a child stdin/stdout.
type stdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	writeMu sync.Mutex
	pending sync.Map // id string -> chan *Response
	nextID  uint64

	doneOnce sync.Once
	done     chan struct{}
	exitErr  atomic.Value // error
}

func newStdioTransport(ctx context.Context, cfg StdioConfig) (*stdioTransport, error) {
	if cfg.Command == "" {
		return nil, errors.New("mcpclient: stdio command required")
	}
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	if len(cfg.Env) > 0 {
		cmd.Env = cfg.Env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpclient: stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpclient: stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpclient: stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcpclient: start: %w", err)
	}
	s := &stdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReaderSize(stdout, 1<<20),
		done:   make(chan struct{}),
	}
	go s.readLoop()
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	return s, nil
}

func (s *stdioTransport) readLoop() {
	defer s.markDone(nil)
	for {
		line, err := s.stdout.ReadBytes('\n')
		if len(line) > 0 {
			var resp Response
			if uerr := json.Unmarshal(line, &resp); uerr == nil && !isNotification(resp.ID) {
				if ch, ok := s.pending.LoadAndDelete(string(resp.ID)); ok {
					ch.(chan *Response) <- &resp
				}
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.markDone(err)
			}
			return
		}
	}
}

func (s *stdioTransport) markDone(err error) {
	s.doneOnce.Do(func() {
		if err != nil {
			s.exitErr.Store(err)
		}
		close(s.done)
	})
}

func (s *stdioTransport) call(ctx context.Context, req *Request) (*Response, error) {
	req.JSONRPC = jsonRPCVersion
	if err := req.validate(); err != nil {
		return nil, err
	}
	if strings.HasPrefix(req.Method, "notifications/") {
		req.ID = nil
		if err := s.write(req); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if len(req.ID) == 0 {
		next := atomic.AddUint64(&s.nextID, 1)
		req.ID = json.RawMessage(fmt.Sprintf("%d", next))
	}
	ch := make(chan *Response, 1)
	s.pending.Store(string(req.ID), ch)
	defer s.pending.Delete(string(req.ID))

	if err := s.write(req); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		if v := s.exitErr.Load(); v != nil {
			return nil, v.(error)
		}
		return nil, errors.New("mcpclient: child exited before response")
	case resp := <-ch:
		return resp, nil
	}
}

func (s *stdioTransport) write(req *Request) error {
	raw, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("mcpclient: marshal: %w", err)
	}
	raw = append(raw, '\n')
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := s.stdin.Write(raw); err != nil {
		return fmt.Errorf("mcpclient: write: %w", err)
	}
	return nil
}

func (s *stdioTransport) close() error {
	_ = s.stdin.Close()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	return nil
}

// pipeTransport is an in-process stdio pair for tests (no subprocess).
type pipeTransport struct {
	stdioTransport
}

func newPipeTransport(r io.Reader, w io.WriteCloser) *pipeTransport {
	s := &pipeTransport{
		stdioTransport: stdioTransport{
			stdin:  w,
			stdout: bufio.NewReaderSize(r, 1<<20),
			done:   make(chan struct{}),
		},
	}
	go s.readLoop()
	return s
}
