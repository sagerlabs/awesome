package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const protocolVersion = "2024-11-05"

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// StdioServer exposes the knowledge MCP adapter over line-delimited JSON-RPC.
type StdioServer struct {
	adapter *Adapter
	in      io.Reader
	out     io.Writer
}

// NewStdioServer creates a minimal MCP-compatible stdio server.
func NewStdioServer(adapter *Adapter, in io.Reader, out io.Writer) *StdioServer {
	return &StdioServer{
		adapter: adapter,
		in:      in,
		out:     out,
	}
}

// Serve reads JSON-RPC messages from stdin-style input and writes responses.
func (s *StdioServer) Serve(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	encoder := json.NewEncoder(s.out)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if err := encoder.Encode(errorResponse(nil, -32700, "parse error")); err != nil {
				return err
			}
			continue
		}

		resp, shouldReply := s.handle(ctx, req)
		if !shouldReply {
			continue
		}
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP stdio: %w", err)
	}
	return nil
}

func (s *StdioServer) handle(ctx context.Context, req jsonRPCRequest) (jsonRPCResponse, bool) {
	if len(req.ID) == 0 {
		return jsonRPCResponse{}, false
	}

	switch req.Method {
	case "initialize":
		return successResponse(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"serverInfo": map[string]any{
				"name":    "tft-knowledge",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		}), true
	case "tools/list":
		return successResponse(req.ID, map[string]any{
			"tools": s.adapter.ListTools(),
		}), true
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid tool call params"), true
		}
		result, err := s.adapter.CallTool(ctx, params.Name, params.Arguments)
		if err != nil {
			return errorResponse(req.ID, -32000, err.Error()), true
		}
		return successResponse(req.ID, toolCallResult{
			Content: []toolContent{
				{
					Type: "text",
					Text: string(result),
				},
			},
		}), true
	default:
		return errorResponse(req.ID, -32601, "method not found"), true
	}
}

func successResponse(id json.RawMessage, result any) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func errorResponse(id json.RawMessage, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
}
