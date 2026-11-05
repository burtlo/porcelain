// Package mcpclient implements a minimal MCP client over line-framed JSON-RPC (stdio).
package mcpclient

import (
	"encoding/json"
	"errors"
	"fmt"
)

const jsonRPCVersion = "2.0"

// Request is a JSON-RPC 2.0 request or notification.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is the standard JSON-RPC error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

func (r *Request) validate() error {
	if r == nil {
		return errors.New("mcpclient: nil request")
	}
	if r.JSONRPC != jsonRPCVersion {
		return fmt.Errorf("mcpclient: bad jsonrpc version %q", r.JSONRPC)
	}
	if r.Method == "" {
		return errors.New("mcpclient: empty method")
	}
	return nil
}

func isNotification(id json.RawMessage) bool {
	return len(id) == 0 || string(id) == "null"
}
