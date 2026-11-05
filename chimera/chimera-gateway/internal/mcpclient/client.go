package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// DefaultProtocolVersion is the MCP protocol version negotiated on initialize.
const DefaultProtocolVersion = "2024-11-05"

// Client speaks MCP over a line-framed JSON-RPC transport.
type Client struct {
	tr        transport
	closeTr   func() error
	handshake sync.Once
	initErr   error
}

type transport interface {
	call(ctx context.Context, req *Request) (*Response, error)
	close() error
}

// Tool is an MCP tool descriptor from tools/list.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

type listToolsResult struct {
	Tools []Tool `json:"tools"`
}

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// NewStdioClient launches cfg.Command and returns a client. Call Handshake before ListTools or CallTool.
func NewStdioClient(ctx context.Context, cfg StdioConfig) (*Client, error) {
	tr, err := newStdioTransport(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{tr: tr, closeTr: tr.close}, nil
}

// NewPipeClient pairs with an in-process MCP server (tests).
func NewPipeClient(serverOut io.Reader, clientIn io.WriteCloser) *Client {
	tr := newPipeTransport(serverOut, clientIn)
	return &Client{tr: tr, closeTr: tr.close}
}

// ChildPID returns the stdio child process ID when running, else 0.
func (c *Client) ChildPID() int {
	if c == nil || c.tr == nil {
		return 0
	}
	st, ok := c.tr.(*stdioTransport)
	if !ok || st.cmd == nil || st.cmd.Process == nil {
		return 0
	}
	return st.cmd.Process.Pid
}

// Close tears down the transport.
func (c *Client) Close() error {
	if c == nil || c.closeTr == nil {
		return nil
	}
	return c.closeTr()
}

// Handshake runs initialize and notifications/initialized per MCP transport requirements.
func (c *Client) Handshake(ctx context.Context) error {
	c.handshake.Do(func() {
		c.initErr = c.doHandshake(ctx)
	})
	return c.initErr
}

func (c *Client) doHandshake(ctx context.Context) error {
	params, err := json.Marshal(map[string]any{
		"protocolVersion": DefaultProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]string{"name": "chimera-gateway", "version": "0.0.0"},
	})
	if err != nil {
		return err
	}
	resp, err := c.tr.call(ctx, &Request{Method: "initialize", Params: params})
	if err != nil {
		return fmt.Errorf("mcpclient: initialize: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("mcpclient: initialize: %w", resp.Error)
	}
	var initRes initializeResult
	if err := json.Unmarshal(resp.Result, &initRes); err != nil {
		return fmt.Errorf("mcpclient: parse initialize: %w", err)
	}
	if initRes.ProtocolVersion == "" {
		return errors.New("mcpclient: initialize: empty protocolVersion")
	}
	_, err = c.tr.call(ctx, &Request{Method: "notifications/initialized"})
	if err != nil {
		return fmt.Errorf("mcpclient: initialized notification: %w", err)
	}
	return nil
}

// ListTools calls tools/list. Handshake must have succeeded.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	if err := c.Handshake(ctx); err != nil {
		return nil, err
	}
	resp, err := c.tr.call(ctx, &Request{Method: "tools/list"})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out listToolsResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, fmt.Errorf("mcpclient: parse tools/list: %w", err)
	}
	return out.Tools, nil
}

// CallTool invokes tools/call with the given JSON arguments (may be nil).
func (c *Client) CallTool(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	if name == "" {
		return nil, errors.New("mcpclient: tool name required")
	}
	if err := c.Handshake(ctx); err != nil {
		return nil, err
	}
	params, err := json.Marshal(struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}{Name: name, Arguments: args})
	if err != nil {
		return nil, err
	}
	resp, err := c.tr.call(ctx, &Request{Method: "tools/call", Params: params})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}
